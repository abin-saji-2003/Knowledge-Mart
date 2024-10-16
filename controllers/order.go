package controllers

import (
	"fmt"
	database "knowledgeMart/config"
	"knowledgeMart/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

//var o_id int

func PlaceOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "user not authorized ",
		})
		return
	}

	userIDStr, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to retrieve user information",
		})
		return
	}

	// addressIDStr := c.Query("addressid")
	// addressID, err := strconv.Atoi(addressIDStr)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"status":  "failed",
	// 		"message": "invalid address ID",
	// 	})
	// 	return
	// }

	var request models.PlaceOrder

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "failed to process request",
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": err.Error(),
		})
		return
	}

	var User models.User

	if err := database.DB.Where("id = ?", userIDStr).First(&User).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "user doesn't exist, please verify user id",
		})
		return
	}

	var CartItems []models.Cart
	var TotalAmount float64
	var sellerID uint

	if err := database.DB.Preload("Product").Where("user_id = ?", userIDStr).Find(&CartItems).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "failed to find the cart",
		})
		return
	}

	if len(CartItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Your cart is empty.",
		})
		return
	}

	for _, item := range CartItems {
		Product := item.Product
		if !Product.Availability {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "failed",
				"message": "items in the cart are out of stock, please update the cart to ensure all items are in stock",
			})
			return
		}
		TotalAmount += Product.Price

		if sellerID == 0 {
			sellerID = Product.SellerID
		} else if sellerID != Product.SellerID {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "failed",
				"message": "You can only add products from one seller to your cart per order.",
			})
			return
		}
	}
	var Address models.Address

	if err := database.DB.Where("user_id = ? AND id = ?", userIDStr, request.AddressID).First(&Address).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"status":  "failed",
			"message": "invalid address, please retry with user's address",
		})
		return
	}

	MethodNo := request.PaymentMethod
	var PaymentMethodOption string
	switch MethodNo {
	case 1:
		//razorpay
		PaymentMethodOption = models.Razorpay
	case 2:
		//wallet
		PaymentMethodOption = models.Wallet
	case 3:
		//COD
		PaymentMethodOption = models.COD
	}

	order := models.Order{
		UserID:        userIDStr,
		TotalAmount:   TotalAmount,
		PaymentMethod: PaymentMethodOption,
		PaymentStatus: models.OrderStatusPending,
		OrderedAt:     time.Now(),
		SellerID:      sellerID,
		Status:        models.OrderStatusPending,
		ShippingAddress: models.ShippingAddress{
			StreetName:   Address.StreetName,
			StreetNumber: Address.StreetNumber,
			City:         Address.City,
			State:        Address.State,
			PinCode:      Address.PinCode,
			PhoneNumber:  Address.PhoneNumber,
		},
	}

	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to create order" + err.Error(),
		})
		return
	}

	//o_id = int(order.OrderID)

	if !CartToOrderItems(userIDStr, order) {
		database.DB.Delete(&order)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to transfer cart items to order",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Order is successfully created",
		"data": gin.H{
			"order_id":      order.OrderID,
			"order_details": order,
		},
	})
}

func CartToOrderItems(UserID uint, Order models.Order) bool {
	var CartItems []models.Cart

	if err := database.DB.Preload("Product").Where("user_id = ?", UserID).Find(&CartItems).Error; err != nil {
		return false
	}

	if len(CartItems) == 0 {
		return false
	}

	// transaction starts
	tx := database.DB.Begin()

	for _, cartItem := range CartItems {
		Product := cartItem.Product

		orderItem := models.OrderItem{
			OrderID:   Order.OrderID,
			ProductID: cartItem.ProductID,
			UserID:    UserID,
			SellerID:  Product.SellerID,
			Price:     Product.Price,
		}

		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			return false
		}
		Product.Availability = false
		if err := tx.Model(&Product).Where("id = ?", Product.ID).Update("availability", Product.Availability).Error; err != nil {
			tx.Rollback()
			return false
		}
	}

	if err := tx.Where("user_id = ? ", UserID).Delete(&CartItems).Error; err != nil {
		tx.Rollback()
		return false
	}

	// transaction ends
	tx.Commit()

	return true

}

func GetUserOrders(c *gin.Context) {
	sellerID, exists := c.Get("sellerID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "seller not authorized",
		})
		return
	}

	sellerIDStr, ok := sellerID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to retrieve seller information",
		})
		return
	}

	var orders []models.Order
	var orderResponses []models.GetSellerOrdersResponse

	if err := database.DB.Where("seller_id = ?", sellerIDStr).Find(&orders).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "no orders found for this seller",
		})
		return
	}

	for _, order := range orders {
		var orderItems []models.OrderItem
		if err := database.DB.Where("order_id = ?", order.OrderID).Find(&orderItems).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "failed",
				"message": "failed to retrieve order items",
			})
			return
		}

		var products []models.ProductArray
		for _, item := range orderItems {
			var product models.Product
			if err := database.DB.Where("id = ?", item.ProductID).First(&product).Error; err == nil {
				products = append(products, models.ProductArray{
					ProductID:   product.ID,
					ProductName: product.Name,
					Description: product.Description,
					Image:       product.Image,
					Price:       product.Price,
					OrderItemID: item.OrderItemID,
				})
			}
		}
		orderResponses = append(orderResponses, models.GetSellerOrdersResponse{
			OrderID:         order.OrderID,
			UserID:          order.UserID,
			SellerID:        order.SellerID,
			PaymentMethod:   order.PaymentMethod,
			PaymentStatus:   order.PaymentStatus,
			TotalAmount:     order.TotalAmount,
			OrderStatus:     order.Status,
			Product:         products,
			ShippingAddress: order.ShippingAddress,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   orderResponses,
	})
}

func SellerUpdateOrderStatus(c *gin.Context) {
	sellerID, exists := c.Get("sellerID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "seller not authorized",
		})
		return
	}

	sellerIDStr, ok := sellerID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to retrieve seller information",
		})
		return
	}

	orderItemId := c.Query("orderitemid")
	if orderItemId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "orderItemId is required",
		})
		return
	}

	var ordersItem models.OrderItem
	if err := database.DB.Where("seller_id = ? AND order_item_id = ?", sellerIDStr, orderItemId).
		First(&ordersItem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "ordersItem not found for this seller",
		})
		return
	}

	var order models.Order
	if err := database.DB.Where("order_id = ?", ordersItem.OrderID).Find(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "order not found",
		})
		return
	}

	// Update the individual order item status
	switch ordersItem.Status {
	case models.OrderStatusPending:
		ordersItem.Status = models.OrderStatusShipped
	case models.OrderStatusShipped:
		ordersItem.Status = models.OrderStatusOutForDelivery
	case models.OrderStatusOutForDelivery:
		ordersItem.Status = models.OrderStatusDelivered
	case models.OrderStatusDelivered:
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Order already delivered",
		})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid order status transition",
		})
		return
	}

	tx := database.DB.Begin()

	// Update the order item status in the transaction
	if err := tx.Model(&ordersItem).Updates(map[string]interface{}{
		"status": ordersItem.Status,
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "Failed to update order item status",
		})
		return
	}

	// Check the statuses of all order items in this order and count them
	var orderItems []models.OrderItem
	if err := tx.Where("order_id = ?", ordersItem.OrderID).Find(&orderItems).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "Failed to retrieve order items",
		})
		return
	}

	// Determine the overall order status based on individual item statuses
	allDelivered := true
	allPending := true
	allConfirmed := true
	allOutforDelivery := true
	for _, item := range orderItems {
		if item.Status != models.OrderStatusDelivered {
			allDelivered = false
		}
		if item.Status != models.OrderStatusPending {
			allPending = false
		}
		if item.Status != models.OrderStatusConfirmed {
			allConfirmed = false
		}
		if item.Status != models.OrderStatusOutForDelivery {
			allOutforDelivery = false
		}
	}

	if allDelivered {
		order.Status = models.OrderStatusDelivered
	} else if allPending {
		order.Status = models.OrderStatusPending
	} else if allConfirmed {
		order.Status = models.OrderStatusConfirmed
	} else if allOutforDelivery {
		order.Status = models.OrderStatusOutForDelivery
	} else {
		order.Status = models.OrderStatusShipped // Set to a reasonable in-between state, such as 'Shipped'
	}

	// Update the overall order status in the transaction
	if err := tx.Model(&order).Updates(map[string]interface{}{
		"status":         order.Status,
		"payment_status": order.PaymentStatus,
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "Failed to update overall order status",
		})
		return
	}

	tx.Commit()

	// Return the updated status
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Order status updated successfully",
		"data": gin.H{
			"newStatus": ordersItem.Status,
		},
	})
}

func UserCheckOrderStatus(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "user not authorized",
		})
		return
	}

	userIDStr, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to retrieve user information",
		})
		return
	}

	var orders []models.Order

	if err := database.DB.Where("user_id = ?", userIDStr).Find(&orders).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "no orders found for this user",
		})
		return
	}

	var userOrderResponses []models.UserOrderResponse
	for _, order := range orders {
		var orderItems []models.OrderItem
		if err := database.DB.Preload("Product").Preload("Seller").Where("order_id = ?", order.OrderID).Find(&orderItems).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "failed",
				"message": "failed to retrieve order items",
			})
			return
		}

		// Initialize status counters
		countPending, countShipped, countDelivered, countOutForDelivery, countConfirmed := 0, 0, 0, 0, 0

		var orderItemResponses []models.OrderItemResponse
		for _, orderItem := range orderItems {
			// Count statuses
			switch orderItem.Status {
			case models.OrderStatusPending:
				countPending++
			case models.OrderStatusShipped:
				countShipped++
			case models.OrderStatusDelivered:
				countDelivered++
			case models.OrderStatusOutForDelivery:
				countOutForDelivery++
			case models.OrderStatusConfirmed:
				countConfirmed++
			}

			// Append individual order item details
			orderItemResponses = append(orderItemResponses, models.OrderItemResponse{
				OrderItemID: orderItem.OrderItemID,
				ProductName: orderItem.Product.Name,
				CategoryID:  orderItem.Product.CategoryID,
				Description: orderItem.Product.Description,
				Price:       orderItem.Price,
				Image:       orderItem.Product.Image,
				SellerName:  orderItem.Seller.UserName,
				OrderStatus: orderItem.Status,
			})
		}

		// Build the status count map, excluding zero counts
		statusCounts := gin.H{}
		if countPending > 0 {
			statusCounts["Pending"] = countPending
		}
		if countShipped > 0 {
			statusCounts["Shipped"] = countShipped
		}
		if countDelivered > 0 {
			statusCounts["Delivered"] = countDelivered
		}
		if countOutForDelivery > 0 {
			statusCounts["OutForDelivery"] = countOutForDelivery
		}
		if countConfirmed > 0 {
			statusCounts["Confirmed"] = countConfirmed
		}

		// Create response for each order, including the filtered non-zero item counts
		userOrderResponses = append(userOrderResponses, models.UserOrderResponse{
			OrderID:         order.OrderID,
			OrderedAt:       order.OrderedAt,
			TotalAmount:     order.TotalAmount,
			Items:           orderItemResponses,
			Status:          order.Status,
			PaymentStatus:   order.PaymentStatus,
			ShippingAddress: order.ShippingAddress,
			ItemCounts:      statusCounts, // Include non-zero status counts
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   userOrderResponses,
	})
}

func CancelOrder(c *gin.Context) {
	sellerID, isSeller := c.Get("sellerID")
	userID, isUser := c.Get("userID")

	if !isSeller && !isUser {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "user or seller not authorized",
		})
		return
	}

	var id uint
	if isSeller {
		id = sellerID.(uint)
	} else if isUser {
		id = userID.(uint)
	}

	orderId := c.Query("orderid")
	itemId := c.Query("itemid")

	if orderId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "orderId is required",
		})
		return
	}

	var orders models.Order
	var condition string
	if isSeller {
		condition = "seller_id = ? AND order_id = ?"
	} else {
		condition = "user_id = ? AND order_id = ?"
	}

	// Retrieve the order
	if err := database.DB.Where(condition, id, orderId).First(&orders).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "order not found for this user or seller",
		})
		return
	}

	// Transaction for handling cancellation and refund
	tx := database.DB.Begin()

	// Cancel single item logic
	if itemId != "" {
		var orderItem models.OrderItem
		if err := tx.Where("order_id = ? AND order_item_id = ?", orderId, itemId).Preload("Product").First(&orderItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "failed",
				"message": "order item not found",
			})
			return
		}

		orderItem.Status = models.OrderStatusCanceled
		if err := tx.Model(&orderItem).Update("status", orderItem.Status).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "failed",
				"message": "failed to cancel order item",
			})
			return
		}

		// Update the total order amount
		orders.TotalAmount -= orderItem.Price
		if err := tx.Model(&orders).Update("total_amount", orders.TotalAmount).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "failed",
				"message": "failed to update order total",
			})
			return
		}

		// Update product availability
		orderItem.Product.Availability = true
		if err := tx.Model(&orderItem.Product).Update("availability", orderItem.Product.Availability).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "failed",
				"message": "failed to update product availability",
			})
			return
		}

		// Refund for single item cancellation
		err := RefundToUser(tx, id, orderId, orderItem.Price, "Single item canceled")
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "failed",
				"message": "failed to refund amount",
			})
			return
		}

		tx.Commit()

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Order item canceled and amount refunded",
		})
		return
	}

	// Cancel entire order logic
	orders.Status = models.OrderStatusCanceled
	orders.PaymentStatus = models.PaymentStatusCanceled

	if err := tx.Model(&orders).Updates(map[string]interface{}{
		"status":         orders.Status,
		"payment_status": orders.PaymentStatus,
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to update order status",
		})
		return
	}

	// Update all items in the order to available
	var orderItems []models.OrderItem
	if err := tx.Preload("Product").Where("order_id = ?", orderId).Find(&orderItems).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "order items not found for this order",
		})
		return
	}

	for _, orderItem := range orderItems {
		orderItem.Product.Availability = true
		if err := tx.Model(&orderItem.Product).Update("availability", orderItem.Product.Availability).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "failed",
				"message": "failed to update product availability",
			})
			return
		}
	}

	// Refund for entire order cancellation
	err := RefundToUser(tx, id, orderId, orders.TotalAmount, "Entire order canceled")
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to refund amount",
		})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Order canceled and amount refunded",
	})
}

// RefundToUser processes the refund by updating the user's wallet
func RefundToUser(tx *gorm.DB, userID uint, orderID string, amount float64, reason string) error {
	var wallet models.UserWallet

	if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}
	newBalance := wallet.CurrentBalance + amount

	walletTransaction := models.UserWallet{
		UserID:          userID,
		WalletPaymentID: fmt.Sprintf("WALLET_%d", time.Now().Unix()), // Generate unique ID
		Type:            "incoming",
		OrderID:         orderID,
		Amount:          amount,
		CurrentBalance:  newBalance,
		Reason:          reason,
	}

	// Save the new wallet transaction and update the balance
	if err := tx.Create(&walletTransaction).Error; err != nil {
		return err
	}

	// Update the current wallet balance
	return tx.Model(&wallet).Update("current_balance", newBalance).Error
}
