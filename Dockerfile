FROM golang:1.23-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Install Go dependencies
RUN go mod tidy

# Build the Go application (surreybank)
RUN go build -o surreybank .

# Command to run the Go app
CMD ["./surreybank"]
