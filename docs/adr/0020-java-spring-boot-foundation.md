# ADR 0020: Java 21 Spring Boot foundation

- Status: Accepted
- Date: 2026-09-01

## Context

The model-neutral Engine needs a Java adapter whose generated projects preserve the
same deterministic Plan and ownership behavior as Go while following Java-native
build, dependency, architecture, and static-analysis practices. A partially emitted
identity or business layer would falsely imply parity and create unsafe AI guesses.

## Decision

1. The Java reference stack is Java 21, Spring Boot 4.1.1, and Maven 3.9 or later.
2. `java-service` `0.1.0` generates deterministic PostgreSQL or MySQL projects with
   Spring Web, JDBC/HikariCP, Flyway database modules, a machine-readable OpenAPI
   foundation, process liveness, and a two-second database readiness check.
3. Runtime database connection values remain environment configuration and are not
   copied into a Blueprint, source file, or generated artifact.
4. `mvn verify` enforces JUnit, ArchUnit, Checkstyle, Spotless, SpotBugs, and Maven
   Enforcer. Constructor injection and meaningful package boundaries are the default.
5. The foundation rejects business modules, capability selections, and frontends
   until each complete vertical slice is implemented and tested for both databases.
6. Generated services have no runtime dependency on Scaffold Agent.

## Consequences

- Java is now a real registered adapter rather than an unavailable target, but its
  support matrix remains explicit and narrower than Go until subsequent M6 slices.
- The chosen stack runs on the local JDK 21/Maven toolchain and on a dedicated CI job.
- Identity, audit, CRUD, OpenAPI expansion, Vue reuse, live database integration,
  and platform capabilities remain additive Java adapter work.
