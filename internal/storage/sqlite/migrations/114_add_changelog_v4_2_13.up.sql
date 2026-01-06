INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.13', '2026-01-06', '## [4.2.13] - 2026-01-06

### Added

- **Maven Dependency Scanner**: Full support for Maven projects
  - Парсинг `pom.xml` с поддержкой properties substitution
  - Сканирование `<dependencies>` и `<dependencyManagement>` блоков
  - Проверка версий через Maven Central (`repo1.maven.org`)
  - Проверка уязвимостей через OSV (ecosystem: Maven)

- **Gradle Dependency Scanner**: Support for Gradle projects (Groovy & Kotlin DSL)
  - Парсинг `build.gradle` (Groovy DSL)
  - Парсинг `build.gradle.kts` (Kotlin DSL)
  - Поддержка различных форматов объявления: string, map syntax
  - Variable substitution из ext блоков
  - Проверка версий через Maven Central

### Technical

- `internal/gitlab/dependencies/parser/maven.go`: New Maven POM parser
- `internal/gitlab/dependencies/parser/gradle.go`: New Gradle parser (Groovy + Kotlin DSL)
- `internal/gitlab/dependencies/registry/maven.go`: Maven Central client
- `internal/gitlab/dependencies/scanner.go`: Integration of Java ecosystem');

