# Knowledge-Mart

KnowledgeMart is an innovative platform for buying and selling second-hand educational items and sharing study notes, built using Go, the Gin framework, PostgreSQL, AWS, and Docker & Kubernetes for containerization and orchestration. Cloudinary is used for note storage. This platform enables users to search for products, manage their profiles, order items, upload and view notes, and much more. The API architecture is containerized and efficiently managed with Kubernetes, ensuring scalability and smooth deployment. It includes dedicated routes for users, sellers, and admins to provide a seamless experience for each group.

## Key Features

- **Comprehensive Search and Filter:**  Quickly find products with advanced search and filter options.

- **Profile and Order Management:** Users can update profiles, track orders, and view order history.

- **Secure Authentication:** Robust user authentication, including OTP verification during signup, ensures secure access.

- **Role-Based Access Control:** The platform provides tailored functionality for users, sellers, and admins, offering optimized interactions and control for each role.

- **Referral and Wallet System:** Users can earn referral rewards and manage wallet balances, providing an incentive-driven experience.

- **Exciting Offers and Coupons:** Users enjoy product discounts, seasonal offers, and redeemable coupons that make learning materials even more affordable. 

- **Payment Integration:** Integrated with Razorpay for secure and seamless order payments.

- **Efficient Note Sharing:**  Cloud-based note upload and sharing, supported by Cloudinary, enables easy access to learning materials.

- **Wallet and Transaction History:** Track referral rewards, purchases, and wallet balances for transparency in transactions.

- **Scalable and Containerized Deployment:** The platform is containerized with Docker and deployed using Kubernetes, ensuring efficient scaling, high availability, and seamless updates.

## Installation

To set up the project locally, follow these steps:

1. **Clone the Repository:**

     ```bash
    git clone https://github.com/spectre2003/Knowledge-Mart.git
    cd Knowledge-Mart
    ```
2. **Set Up the Environment Variables:**

    Create a `.env` file in the root directory and add the following variables:

    ```bash
    DB_HOST=127.0.0.1
    DB_USER=your_database_username
    DB_PASSWORD=your_database_password
    DB_NAME=your_database_name
    DB_PORT=5432
    DB_SSLMODE=your_database_sslmode
    SMTPAPP=your_smtp_app_password
    JWTSECRET=your_jwt_secret_key
    SERVERIP=localhost:8080
    CLIENTID=your_google_auth_client_id
    CLIENTSECRET=your_google_oauth_client_secret
    CLOUDNAME=your_cloudinary_cloud_name
    CLOUDINARYACCESSKEY=your_cloudinary_access_key
    CLOUDINARYSECRETKEY=your_cloudinary_secret_key
    CLOUDINARYURL=your_cloudinary_url
    RAZORPAY_KEY_ID=your_razorpay_key_id
    RAZORPAY_KEY_SECRET=your_razorpay_key_secret
    ```

3. **Install Dependencies:**

    ```bash
    go mod tidy
    ```

4. **Run the Application:**

    ```bash
    go run .
    ```

## Run with Docker

 **Using Docker CLI:**

    If your project has docker-compose.yml, you can run:

    ```bash
    docker-compose up -d
    ```

## Option 3: Deploy with Kubernetes

1. **Start Minikube (If Running Locally):**

    ```bash
    minikube start
    ```

2. **Apply Deployments and Services:**

    ```bash
    kubectl apply -f k8s/deployment-1.yaml
    kubectl apply -f k8s/deployment-2.yaml
    kubectl apply -f k8s/service-1.yaml
    kubectl apply -f k8s/service-2.yaml
    ```

3. **Configure Ingress (For Routing Traffic):**

    ```bash
    kubectl apply -f k8s/ingress.yaml
    ```

4. ** Verify Deployment:**

    ```bash
    kubectl get pods
    kubectl get services
    kubectl get ingress
    ```


## API Documentation

Detailed API documentation is available [here](https://documenter.getpostman.com/view/38480579/2sAY4x9M3Y).

---