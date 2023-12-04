# The Artwork seach page

```mermaid
sequenceDiagram
    actor U as User
    participant B as Browser
    participant S as Server
    U->>B: Open search page
    B->>S: Request search page
    S->>B: Return search page
    B->>U: Display search page
    B->>S: Get unfilled search result page
    S->>B: Return unfilled search result page
    U->>B: Enter search criteria
    B->>S: Request search
    S->>B: Return search results
    B->>U: Display search results
```
