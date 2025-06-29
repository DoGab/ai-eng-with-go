package quiz

import (
	"log"
	"math/rand"
	"strings"
)

var hardcodedContent = map[string][]string{
	"database": {
		"Database indexing is crucial for query performance. B-tree indexes are most common, providing O(log n) lookup time. However, they add overhead to write operations as indexes must be maintained during INSERT, UPDATE, and DELETE operations.",
		"Database normalization reduces data redundancy by organizing data into related tables. First Normal Form (1NF) eliminates repeating groups, Second Normal Form (2NF) removes partial dependencies, and Third Normal Form (3NF) eliminates transitive dependencies.",
		"ACID properties ensure database transaction reliability: Atomicity (all or nothing), Consistency (valid state transitions), Isolation (concurrent transactions don't interfere), and Durability (committed changes persist).",
		"Database sharding distributes data across multiple servers horizontally. Common sharding strategies include range-based (by key ranges), hash-based (by hash function), and directory-based (using a lookup service).",
	},
	"performance": {
		"Caching strategies improve application performance by storing frequently accessed data in memory. Common patterns include cache-aside (application manages cache), write-through (write to cache and database simultaneously), and write-behind (async write to database).",
		"Load balancing distributes incoming requests across multiple servers. Algorithms include round-robin, least connections, weighted round-robin, and IP hash. Health checks ensure requests aren't sent to failing servers.",
		"Database connection pooling reduces overhead by reusing connections. Pool size should be tuned based on concurrent users and database capacity. Too few connections create bottlenecks, too many can overwhelm the database.",
		"Query optimization involves analyzing execution plans, adding proper indexes, avoiding N+1 queries, and using database-specific optimizations. EXPLAIN ANALYZE shows actual query performance metrics.",
	},
	"scalability": {
		"Horizontal scaling adds more servers to handle increased load, while vertical scaling upgrades existing hardware. Horizontal scaling provides better fault tolerance but requires load balancing and data distribution strategies.",
		"Microservices architecture breaks applications into small, independent services. Benefits include technology diversity, team autonomy, and independent scaling. Challenges include distributed system complexity and inter-service communication.",
		"Event-driven architecture uses events to trigger and communicate between services. Patterns include event sourcing (storing events as primary data), CQRS (separate read/write models), and event streaming with tools like Kafka.",
		"Auto-scaling automatically adjusts resources based on demand. Metrics include CPU utilization, memory usage, request rate, and custom business metrics. Policies define scale-up/down thresholds and cooldown periods.",
	},
	"caching": {
		"Redis is an in-memory data store supporting various data structures: strings, hashes, lists, sets, sorted sets, and streams. It provides persistence options including RDB snapshots and AOF logging.",
		"Cache invalidation strategies ensure data consistency. Time-based expiration (TTL) is simple but may serve stale data. Cache-aside pattern gives application control over when to invalidate specific keys.",
		"CDN (Content Delivery Network) caches static content at edge locations worldwide. This reduces latency by serving content from locations closer to users. Dynamic content can also be cached with proper cache headers.",
		"Application-level caching can occur at multiple layers: browser cache, reverse proxy cache (nginx, Varnish), application cache (in-memory objects), and database query cache. Each layer has different invalidation strategies.",
	},
	"system": {
		"CAP theorem states distributed systems can guarantee at most two of: Consistency (all nodes see same data), Availability (system remains operational), and Partition tolerance (system continues despite network failures).",
		"Consistent hashing distributes data across nodes in a way that minimizes reshuffling when nodes are added or removed. Virtual nodes improve load distribution and fault tolerance.",
		"Circuit breaker pattern prevents cascading failures in distributed systems. It monitors failure rates and 'opens' to prevent calls to failing services, allowing them time to recover.",
		"Eventual consistency allows distributed systems to be temporarily inconsistent but guarantees convergence. This enables high availability and partition tolerance at the cost of immediate consistency.",
	},
	"design": {
		"Design patterns provide reusable solutions to common problems. Creational patterns (Singleton, Factory) deal with object creation. Structural patterns (Adapter, Decorator) deal with object composition. Behavioral patterns (Observer, Strategy) deal with communication.",
		"Domain-driven design focuses on the core domain and domain logic. Key concepts include entities (objects with identity), value objects (immutable objects), aggregates (consistency boundaries), and bounded contexts (logical boundaries).",
		"API design principles include REST (Representational State Transfer) with proper HTTP methods, resource-based URLs, and stateless communication. GraphQL provides flexible querying but adds complexity.",
		"Database design involves choosing between SQL (ACID compliance, complex queries) and NoSQL (scalability, flexibility). Consider data structure, query patterns, consistency requirements, and scaling needs.",
	},
}

func GetContentForTopics(topics []string) []string {
	log.Printf("[INFO] Starting content retrieval for topics: %v", topics)
	var content []string
	var matchedKeys []string
	
	for _, topic := range topics {
		topicLower := strings.ToLower(topic)
		log.Printf("[INFO] Processing topic: '%s' (normalized: '%s')", topic, topicLower)
		
		for key, values := range hardcodedContent {
			if strings.Contains(topicLower, key) {
				log.Printf("[INFO] Topic '%s' matched content key '%s', adding %d content chunks", topic, key, len(values))
				content = append(content, values...)
				matchedKeys = append(matchedKeys, key)
				
				for i, chunk := range values {
					log.Printf("[INFO] Content chunk %d for key '%s': %.100s...", i+1, key, chunk)
				}
			}
		}
	}
	
	if len(content) == 0 {
		log.Printf("[INFO] No content matched for topics %v, using fallback general content", topics)
		fallbackContent := []string{
			"General software engineering principles include separation of concerns, single responsibility, and loose coupling. Code should be readable, maintainable, and testable.",
			"Software architecture patterns include layered architecture (presentation, business, data), hexagonal architecture (ports and adapters), and clean architecture (dependency inversion).",
			"Testing strategies include unit tests (individual components), integration tests (component interactions), and end-to-end tests (full user workflows). Test-driven development (TDD) writes tests before code.",
		}
		content = append(content, fallbackContent...)
		
		for i, chunk := range fallbackContent {
			log.Printf("[INFO] Fallback content chunk %d: %.100s...", i+1, chunk)
		}
	} else {
		log.Printf("[INFO] Successfully matched content from keys: %v", matchedKeys)
	}
	
	log.Printf("[INFO] Total content chunks before shuffle: %d", len(content))
	shuffleStrings(content)
	
	originalCount := len(content)
	if len(content) > 5 {
		content = content[:5]
		log.Printf("[INFO] Truncated content from %d to %d chunks", originalCount, len(content))
	}
	
	log.Printf("[INFO] Final content chunks being returned (%d total):", len(content))
	for i, chunk := range content {
		log.Printf("[INFO] Final chunk %d: %.150s...", i+1, chunk)
	}
	
	return content
}

func shuffleStrings(slice []string) {
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}