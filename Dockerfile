# Start from golang base image
FROM golang:alpine3.18 as builder

# Set ARG for git credentials
ARG GIT_TOKEN
ARG GIT_USER

# Install Git
RUN apk --no-cache add git

# Set the Current Working Directory inside the container
WORKDIR /app

# Configure git to use the supplied token
RUN git config --global url."https://oauth2:${GIT_TOKEN}@github.com/".insteadOf "https://github.com/"

# Copy go mod and sum files
COPY go.mod go.sum ./

# Set GOPRIVATE env var
RUN go env -w GOPRIVATE=github.com/$GIT_USER/*

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

######## Start a new stage from scratch #######
FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Expose port 9443 on the container
EXPOSE 9443

# Command to run the executable
CMD ["./main"] 
