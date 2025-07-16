# itsjustintv Workflow Diagrams

## 1. High-Level System Architecture

```mermaid
graph TB
    subgraph "External Systems"
        T[Twitch EventSub API]
        D[Downstream Webhooks]
        F[File System]
    end
    
    subgraph "itsjustintv Service"
        H[HTTP Server]
        C[Config Manager]
        TC[Twitch Client]
        WD[Webhook Dispatcher]
        E[Metadata Enricher]
        CM[Cache Manager]
        RM[Retry Manager]
        OW[Output Writer]
        TM[Telemetry Manager]
    end
    
    T -->|POST /twitch| H
    H --> TC
    H --> WD
    WD --> E
    WD --> CM
    WD --> RM
    WD --> D
    WD --> OW
    OW --> F
    C -.->|config| H
    C -.->|config| TC
    C -.->|config| WD
    TM -.->|observability| H
    TM -.->|observability| WD
    
    style T fill:#e1f5fe
    style D fill:#e8f5e8
    style F fill:#fff3e0
```

## 2. Webhook Receipt and Processing Flow

```mermaid
sequenceDiagram
    participant T as Twitch EventSub
    participant H as HTTP Server
    participant V as Webhook Validator
    participant P as Twitch Processor
    participant CM as Cache Manager
    participant E as Enricher
    participant WD as Webhook Dispatcher
    participant D as Downstream Webhook
    participant OW as Output Writer
    
    T->>H: POST /twitch (stream.online)
    H->>V: Validate HMAC signature
    V-->>H: Valid ✓
    H->>P: Process notification
    P->>P: Match streamer config
    P->>P: Check tag filters
    P-->>H: Processed event
    H->>CM: Check for duplicates
    CM-->>H: Not duplicate ✓
    H->>E: Enrich metadata
    E->>E: Fetch profile image
    E->>E: Get view/follower count
    E->>E: Merge tags
    E-->>H: Enriched payload
    H->>WD: Dispatch webhook
    WD->>D: POST webhook payload
    D-->>WD: 200 OK
    WD->>OW: Write to output file
    H-->>T: 200 OK
    
    Note over H,WD: If webhook fails, goes to retry queue
```

## 3. Service Startup and Subscription Management

```mermaid
flowchart TD
    A[Service Startup] --> B[Load Configuration]
    B --> C[Validate Config]
    C --> D[Start Twitch Client]
    D --> E[Resolve User IDs]
    E --> F[Start Other Components]
    F --> G[Setup HTTP Routes]
    G --> H[Configure HTTP Server]
    H --> I[Setup TLS if enabled]
    I --> J[Start HTTP Server<br/>in goroutine]
    J --> K[Wait 100ms for server<br/>to start listening]
    K --> L[Start Subscription Manager]
    L --> M[Fetch Current Subscriptions]
    M --> N[Compare with Config]
    N --> O{Missing Subscriptions?}
    O -->|Yes| P[Register Missing Subs<br/>✅ Server is running!]
    O -->|No| Q[Start Background Tasks]
    P --> Q
    Q --> R[Service Running]
    
    subgraph "Background Tasks"
        S[Subscription Validation<br/>Every 1h + 15min splay]
        T[Config File Watcher<br/>Hot reload on changes]
        U[Retry Queue Processor<br/>Exponential backoff]
    end
    
    R --> S
    R --> T
    R --> U
    
    subgraph "Fixed Race Condition"
        V[✅ HTTP server starts FIRST]
        W[✅ Then subscriptions are created]
        X[✅ Twitch can verify webhook endpoint]
    end
    
    style A fill:#e3f2fd
    style R fill:#e8f5e8
    style S fill:#fff3e0
    style T fill:#fff3e0
    style U fill:#fff3e0
    style V fill:#e8f5e8
    style W fill:#e8f5e8
    style X fill:#e8f5e8
```

## 4. Configuration Hot-Reload Workflow

```mermaid
flowchart LR
    A[config.toml] -->|File Change| B[fsnotify Watcher]
    B --> C[Parse New Config]
    C --> D{Valid Config?}
    D -->|No| E[Log Error<br/>Keep Old Config]
    D -->|Yes| F[Apply New Config]
    F --> G[Update Twitch Client]
    G --> H[Refresh Subscriptions]
    H --> I[Update Webhook Dispatcher]
    I --> J[Update Other Components]
    J --> K[Config Reload Complete]
    
    E --> L[Continue with Old Config]
    
    style A fill:#e1f5fe
    style D fill:#fff9c4
    style E fill:#ffebee
    style K fill:#e8f5e8
    style L fill:#fff3e0
```

## 5. Retry Logic and Error Handling

```mermaid
stateDiagram-v2
    [*] --> WebhookDispatch
    WebhookDispatch --> Success: HTTP 2xx
    WebhookDispatch --> Failed: HTTP 4xx/5xx or Timeout
    
    Success --> WriteOutput
    WriteOutput --> [*]
    
    Failed --> RetryQueue: Add to queue
    RetryQueue --> WaitBackoff: Exponential backoff
    WaitBackoff --> RetryAttempt: 30s, 1m, 2m, 4m...
    RetryAttempt --> Success: HTTP 2xx
    RetryAttempt --> Failed: Still failing
    RetryAttempt --> MaxRetries: Attempts exhausted
    
    MaxRetries --> DeadLetter: Log failure
    DeadLetter --> [*]
    
    note right of WaitBackoff
        Initial: 30s
        Max: 30min
        Factor: 2.0
    end note
```

## 6. Cache and Deduplication Strategy

```mermaid
graph TB
    subgraph "Event Processing"
        A[Incoming Stream Event]
        B[Generate Event Key<br/>user_id + stream_id + started_at]
        C{Duplicate Check}
        D[Process Event]
        E[Skip Event]
        F[Add to Cache<br/>TTL: 2h]
    end
    
    subgraph "Cache Storage"
        G[In-Memory Cache]
        H[Persistent JSON File]
        I[Profile Image Cache<br/>TTL: 7 days]
    end
    
    A --> B
    B --> C
    C -->|Not Found| D
    C -->|Found| E
    D --> F
    F --> G
    G -.->|Persist| H
    D --> I
    
    style C fill:#fff9c4
    style E fill:#ffebee
    style D fill:#e8f5e8
```

## 7. Metadata Enrichment Pipeline

```mermaid
flowchart LR
    A[Stream Event] --> B[Base Payload Creation]
    B --> C[Profile Image Fetch]
    C --> D{Image Cached?}
    D -->|Yes| E[Use Cached Image]
    D -->|No| F[Fetch from Twitch API]
    F --> G[Cache Image<br/>7 days TTL]
    G --> E
    E --> H[Get View Count]
    H --> I[Get Follower Count]
    I --> J[Fetch Stream Tags]
    J --> K[Merge with Config Tags]
    K --> L[Detect Language]
    L --> M{Tag Filter Match?}
    M -->|No| N[Block Webhook]
    M -->|Yes| O[Enriched Payload Ready]
    
    style D fill:#fff9c4
    style M fill:#fff9c4
    style N fill:#ffebee
    style O fill:#e8f5e8
```

## 8. TLS Certificate Management

```mermaid
stateDiagram-v2
    [*] --> CheckConfig
    CheckConfig --> TLSDisabled: TLS not enabled
    CheckConfig --> CheckCerts: TLS enabled
    
    TLSDisabled --> HTTPOnly
    HTTPOnly --> [*]
    
    CheckCerts --> ValidCerts: Certs exist & valid
    CheckCerts --> NeedCerts: No certs or expired
    
    ValidCerts --> ScheduleRenewal
    ScheduleRenewal --> HTTPSReady
    
    NeedCerts --> StartACME
    StartACME --> ChallengeServer: Start HTTP-01 on :80
    ChallengeServer --> GetCerts: Request from Let's Encrypt
    GetCerts --> StoreCerts: Save to disk
    StoreCerts --> HTTPSReady
    
    HTTPSReady --> [*]
    
    note right of ScheduleRenewal
        Background renewal
        before expiration
    end note
```

## 9. Component Interaction Overview

```mermaid
graph TB
    subgraph "Core Services"
        HS[HTTP Server<br/>:8080]
        CM[Config Manager<br/>TOML + ENV]
        TC[Twitch Client<br/>OAuth + EventSub]
    end
    
    subgraph "Processing Pipeline"
        WV[Webhook Validator<br/>HMAC Check]
        TP[Twitch Processor<br/>Event Parsing]
        ME[Metadata Enricher<br/>API Calls]
        WD[Webhook Dispatcher<br/>HTTP Client]
    end
    
    subgraph "Storage & Persistence"
        CA[Cache Manager<br/>Memory + JSON]
        RM[Retry Manager<br/>Exponential Backoff]
        OW[Output Writer<br/>FIFO Queue]
    end
    
    subgraph "Observability"
        TM[Telemetry Manager<br/>OpenTelemetry]
        LG[Structured Logging<br/>JSON Format]
    end
    
    HS --> WV
    WV --> TP
    TP --> ME
    ME --> WD
    WD --> CA
    WD --> RM
    WD --> OW
    
    CM -.->|config| HS
    CM -.->|config| TC
    CM -.->|config| WD
    
    TM -.->|metrics| HS
    TM -.->|metrics| WD
    TM -.->|traces| ME
    
    LG -.->|logs| HS
    LG -.->|logs| WD
    LG -.->|logs| RM
    
    style HS fill:#e3f2fd
    style WD fill:#e8f5e8
    style CA fill:#fff3e0
    style TM fill:#f3e5f5
```

## 10. Deployment Architecture Options

```mermaid
graph TB
    subgraph "Option 1: Direct HTTPS"
        A1[itsjustintv<br/>:443 HTTPS]
        A2[Let's Encrypt<br/>Auto-cert]
        A3[Twitch EventSub]
        
        A3 -->|HTTPS| A1
        A1 -.->|ACME| A2
    end
    
    subgraph "Option 2: Reverse Proxy"
        B1[nginx/traefik<br/>:443 HTTPS]
        B2[itsjustintv<br/>:8080 HTTP]
        B3[Twitch EventSub]
        
        B3 -->|HTTPS| B1
        B1 -->|HTTP| B2
    end
    
    subgraph "Option 3: Container"
        C1[Docker Container]
        C2[itsjustintv binary]
        C3[config.toml]
        C4[data/ volume]
        
        C1 --> C2
        C1 --> C3
        C1 --> C4
    end
    
    style A1 fill:#e3f2fd
    style B1 fill:#e8f5e8
    style B2 fill:#e3f2fd
    style C1 fill:#fff3e0
```

These diagrams illustrate the complete workflow and architecture of the itsjustintv service, showing how components interact and data flows through the system.