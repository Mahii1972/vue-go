package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/mailgun/mailgun-go/v4"
)

// Config holds the application configuration
type Config struct {
	Domain    string
	ApiKey    string
	FromName  string
	FromEmail string
}

// EmailService handles all email related operations
type EmailService struct {
	mg     *mailgun.MailgunImpl
	config Config
}

// ProductEmail represents the product email request
type ProductEmail struct {
	ProductName    string  `json:"product_name"`
	Price          float64 `json:"price"`
	Description    string  `json:"description"`
	RecipientEmail string  `json:"email"`
}

// NewEmailService creates a new email service instance
func NewEmailService(config Config) *EmailService {
	return &EmailService{
		mg:     mailgun.NewMailgun(config.Domain, config.ApiKey),
		config: config,
	}
}

// SendProductEmail sends product details via email
func (s *EmailService) SendProductEmail(ctx context.Context, data ProductEmail) (string, string, error) {
	emailBody := s.formatProductEmail(data)
	sender := fmt.Sprintf("%s <%s@%s>", s.config.FromName, s.config.FromEmail, s.config.Domain)

	message := mailgun.NewMessage(
		sender,
		"Product Information",
		emailBody,
		data.RecipientEmail,
	)

	return s.mg.Send(ctx, message)
}

// formatProductEmail formats the email body
func (s *EmailService) formatProductEmail(data ProductEmail) string {
	return fmt.Sprintf(`
Product Details:
---------------
Name: %s
Price: $%.2f
Description: %s
`,
		data.ProductName,
		data.Price,
		data.Description,
	)
}

// Handler represents the HTTP handler dependencies
type Handler struct {
	emailService *EmailService
}

// NewHandler creates a new handler instance
func NewHandler(emailService *EmailService) *Handler {
	return &Handler{
		emailService: emailService,
	}
}

// SendProductHandler handles the product email endpoint
func (h *Handler) SendProductHandler(c *gin.Context) {
	var productData ProductEmail
	if err := c.BindJSON(&productData); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*10)
	defer cancel()

	resp, id, err := h.emailService.SendProductEmail(ctx, productData)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to send email"})
		return
	}

	c.JSON(200, gin.H{
		"message":  "Email sent successfully",
		"id":       id,
		"response": resp,
	})
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	config := Config{
		Domain:    os.Getenv("MAILGUN_DOMAIN"),
		ApiKey:    os.Getenv("MAILGUN_API_KEY"),
		FromName:  os.Getenv("MAILGUN_FROM_NAME"),
		FromEmail: os.Getenv("MAILGUN_FROM_EMAIL"),
	}

	// Initialize services and handlers
	emailService := NewEmailService(config)
	handler := NewHandler(emailService)

	// Setup router with CORS
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.POST("/send-product", handler.SendProductHandler)

	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
