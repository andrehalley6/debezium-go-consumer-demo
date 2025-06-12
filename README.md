# Debezium Consumer Demo (Go + PostgreSQL + Kafka)

This project demonstrates how to consume PostgreSQL change events using **Debezium**, **Kafka**, and a **Go 1.23 consumer**.

---

## 🧱 Tech Stack

- Go 1.23 (with confluent-kafka-go)
- Kafka (via Apache Kafka)
- Debezium PostgreSQL Connector
- PostgreSQL (with logical replication enabled)
- Docker & Docker Compose

---

## 🛠️ Prerequisites

- [Docker](https://www.docker.com/)
- [Go 1.23](https://go.dev/dl/)

---

## 🚀 Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/your-username/debezium-go-consumer-demo.git
cd debezium-go-consumer-demo
```

### 2. Start services with Docker Compose

```bash
docker-compose -f docker-compose.yaml up -d
```

This will run all services (except Go service) in Docker

### 3. Create Table and Enable Logical Replication on PostgreSQL

#### Create table products and add row data

```bash
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    price DECIMAL
);
INSERT INTO products (name, price) VALUES ('Product A', 100.00);
```

```bash
docker exec -it postgres-debezium bash
psql -U admin -d debezium-demo
```

#### Inside the PostgreSQL terminal, run:

```bash
ALTER SYSTEM SET wal_level = logical;
SELECT pg_reload_conf();
```

#### Restart PostgreSQL container
```bash
docker restart postgres-debezium
```

### 4. Register the Debezium Connector

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "postgres-products-connector",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "postgres-debezium",
      "database.port": "5432",
      "database.user": "admin",
      "database.password": "password",
      "database.dbname": "debezium-demo",
      "database.server.name": "pgdemo",
      "plugin.name": "pgoutput",
      "table.include.list": "public.products",
      "slot.name": "products_slot",
      "topic.prefix": "pgdemo"
    }
  }'
```

Once successful, you'll receive a JSON confirmation of the connector.

### 5. Verify Kafka Topics

```bash
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list
```

Search for:
```bash
pgdemo.public.products
```

### 6. Run Go Consumer

Run Go app
```bash
go mod tidy
go run main.go
```

### 7. Test the Go consumer

Try inserting, updating, or deleting data in the `products` table

### 8. Cleanup

```bash
docker-compose down -v
```