FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod .

RUN go mod download

COPY . .

RUN go build -o verify_files main.go

EXPOSE 8080

ENV SECRET_KEY=my_secret_key
ENV PORT=8080

CMD ["./verify_files"]
