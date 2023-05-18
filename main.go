package main

import (
	"context"
	"log"
	"os"

	"github.com/SigNoz/sample-golang-app/controllers"
	"github.com/SigNoz/sample-golang-app/models"
	"github.com/joho/godotenv"
	"google.golang.org/grpc/credentials"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func getEnvFromFile(envName string) string {
	err := godotenv.Load(".env")
	if err != nil {
		return err.Error()
	}
	return os.Getenv(envName)
}

var (
	serviceName  = getEnvFromFile("SERVICE_NAME")
	collectorURL = getEnvFromFile("OTEL_EXPORTER_OTLP_ENDPOINT")
	insecure     = getEnvFromFile("INSECURE_MODE")
)

func initTracer() func(context.Context) error {

	secureOption := otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	if len(insecure) > 0 {
		secureOption = otlptracegrpc.WithInsecure()
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(collectorURL),
		),
	)

	if err != nil {
		log.Fatal(err)
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		log.Fatal("Could not set resources: ", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)
	return exporter.Shutdown
}

func main() {

	cleanup := initTracer()
	defer cleanup(context.Background())

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	log.Println(serviceName, collectorURL, insecure)
	r.Use(otelgin.Middleware(serviceName))
	// Connect to database
	models.ConnectDatabase()

	controllers.CreateDummyBooks()
	// Routes
	r.GET("/books", controllers.FindBooks)
	r.GET("/books/:id", controllers.FindBook)
	r.POST("/books", controllers.CreateBook)
	r.PATCH("/books/:id", controllers.UpdateBook)
	r.DELETE("/books/:id", controllers.DeleteBook)

	// Run the server
	r.Run(":6000")
}
