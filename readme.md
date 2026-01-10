# 🚀 High-Throughput Distributed Analytics Engine

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Enabled-2496ED?logo=docker&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-Queue-DC382D?logo=redis&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-NeonDB-336791?logo=postgresql&logoColor=white)
![k6](https://img.shields.io/badge/k6-Load%20Tested-7D64FF?logo=k6&logoColor=white)

A high-concurrency event ingestion pipeline designed to handle massive traffic spikes. Built with **Golang**, this system uses an **asynchronous architecture** (API → Redis → Worker) to achieve sub-millisecond API latency while ensuring data consistency in **PostgreSQL**.

---

## 🏗️ Architecture

The system decouples **ingestion** from **processing** to ensure high availability and write speeds.

```mermaid
graph LR
    User(Client / k6) -- HTTP POST --> LB[Load Balancer]
    LB --> API1[Go API Replica 1]
    LB --> API2[Go API Replica 2]
    LB --> API3[Go API Replica 3]
    API1 -- Async Push --> Redis[(Redis Queue)]
    API2 -- Async Push --> Redis
    API3 -- Async Push --> Redis
    Redis -- Pop Batch --> Worker1[Go Background Worker]
    Redis -- Pop Batch --> Worker2[Go Background Worker]
    Redis -- Pop Batch --> Worker3[Go Background Worker]
    Worker1 -- Bulk Insert --> DB[(Neon PostgreSQL
    Raw Data Store - 24 hr)]
    Worker2 -- Bulk Insert --> DB[(Neon PostgreSQL
    Raw Data Store - 24 hr)]
    Worker3 -- Bulk Insert --> DB[(Neon PostgreSQL
    Raw Data Store - 24 hr)]
    Cron2[Delete Cron Job] -- Clean up Cron Job --> DB
    DB -- Cron Job --> Cron
    Cron -- Aggregation --> DB2[(Hourly)]
    Cron -- Aggregation --> DB3[(Monthly)]
    Cron -- Aggregation --> DB4[(Daily)]
    Cron -- Aggregation --> DB5[(Yearly)]   
    
```

### Key Components
- Ingestion API (Go): Lightweight HTTP server. Accepts JSON events, validates them, and pushes them instantly to a Redis List. Zero database blocking.
- Message Queue (Redis): Acts as a shock absorber. Handles traffic spikes (e.g., 2k+ events/sec) without overwhelming the database.
- Worker Service (Go): Consumes messages from Redis, batches them (e.g., 2000 events/batch), and performs efficient bulk inserts into PostgreSQL.
- Cron Service: Handles periodic aggregation and cleanup tasks.

### Performance BenchmarksTested on a local single-node environment (Consumer Laptop) using k6.

| Metric | Single Instance | 3-Node Cluster (Optimized) | Description |
| :--- | :--- | :--- | :--- |
| **Throughput** | 1,578 Req/Sec | **1,626 Req/Sec** | Sustained load over 2 minutes. |
| **Latency (p95)** | 5.45 ms | **1.23 ms** ⚡ | 95% of requests completed in < 1.3ms. |
| **Reliability** | 100% | **100%** | Zero dropped requests under load. |
| **Capacity** | 136M / Day | **140M+ / Day** | Extrapolated daily volume. |
    
    
    
## Tech Stack
- **Language**: Golang (1.21+) 
- **Database**: PostgreSQL (via Neon Serverless)
- **Queue**: Redis 
- **Containerization**: Docker & Docker Compose 
- **Testing**: Grafana k6

## Project Structure
``` bash
.
├── cmd/
│   ├── api/          
│   ├── worker/      
│   └── cron/         
├── internal/
│   └── queue/        
├── docker-compose.yml 
├── Dockerfile        
└── loadtest.js       
```

## Future Improvements
- **Batching**: Implement dynamic batching in the API layer for even higher throughput.
- **Partitioning**: Use PostgreSQL partitioning for time-series data management.

Author
Agniva Sengupta Building scalable systems one goroutine at a time.