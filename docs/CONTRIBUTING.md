# Contributing to QuietHire

Thank you for your interest in contributing to QuietHire! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)
- [Project Structure](#project-structure)
- [Testing](#testing)

---

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inclusive environment for all contributors. We expect respectful and professional conduct from everyone participating in this project.

### Expected Behavior

- Use welcoming and inclusive language
- Be respectful of differing viewpoints and experiences
- Accept constructive criticism gracefully
- Focus on what is best for the project and community
- Show empathy towards other community members

### Unacceptable Behavior

- Harassment, trolling, or discriminatory comments
- Personal or political attacks
- Publishing others' private information without permission
- Any conduct that could reasonably be considered inappropriate

---

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- Go 1.21 or higher
- Python 3.12 or higher
- Docker and Docker Compose
- Git
- A GitHub account

### Familiarize Yourself with the Project

1. Read the [README.md](../README.md)
2. Review the [Architecture Documentation](architecture.md)
3. Understand the [API Reference](api.md)
4. Browse existing issues and pull requests

---

## Development Setup

### 1. Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/quiethire.git
cd quiethire

# Add upstream remote
git remote add upstream https://github.com/itisrohit/quiethire.git
```

### 2. Set Up Environment

```bash
# Copy environment file
cp .env.example .env

# Edit .env with your configuration
# At minimum, set:
# - DB_PASSWORD
# - CLICKHOUSE_PASSWORD
# - TYPESENSE_API_KEY
```

### 3. Start Services

```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps
```

### 4. Verify Setup

```bash
# API health check
curl http://localhost:3000/health

# Check logs
docker-compose logs -f api worker
```

---

## How to Contribute

### Types of Contributions

We welcome:

- **Bug fixes** - Fix issues reported in GitHub Issues
- **Feature enhancements** - Improve existing functionality
- **New features** - Add new capabilities (discuss in issue first)
- **Documentation** - Improve or add documentation
- **Performance improvements** - Optimize code or database queries
- **Tests** - Add or improve test coverage
- **Code quality** - Refactoring, linting fixes

### Contribution Workflow

1. **Find or create an issue**
   - Check existing issues for something to work on
   - For new features, create an issue first to discuss

2. **Create a branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-number-description
   ```

3. **Make your changes**
   - Write clean, readable code
   - Follow coding standards (see below)
   - Add tests for new functionality
   - Update documentation as needed

4. **Test your changes**
   ```bash
   # Go tests
   cd apps/api && go test ./...
   
   # Python tests
   cd apps/parser && uv run pytest
   
   # Linting
   cd apps/api && golangci-lint run
   cd apps/parser && uv run ruff check .
   ```

5. **Commit your changes**
   - Follow commit guidelines (see below)
   - Sign your commits (optional but recommended)

6. **Push and create pull request**
   ```bash
   git push origin feature/your-feature-name
   ```
   - Create PR on GitHub
   - Fill out the PR template
   - Link related issues

---

## Coding Standards

### Go Code Standards

**Style Guide**: Follow [Effective Go](https://go.dev/doc/effective_go)

**Key Conventions**:
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Keep functions small and focused
- Use meaningful variable names
- Add comments for exported functions
- Handle errors explicitly

**Example**:
```go
// GetJobByID retrieves a job from ClickHouse by its UUID
func (s *Service) GetJobByID(ctx context.Context, id string) (*Job, error) {
    if id == "" {
        return nil, fmt.Errorf("job ID cannot be empty")
    }
    
    query := "SELECT * FROM jobs WHERE id = ?"
    var job Job
    if err := s.db.QueryRow(ctx, query, id).Scan(&job); err != nil {
        return nil, fmt.Errorf("failed to query job: %w", err)
    }
    
    return &job, nil
}
```

### Python Code Standards

**Style Guide**: Follow [PEP 8](https://peps.python.org/pep-0008/)

**Key Conventions**:
- Use `ruff` for linting and formatting
- Use `mypy` for type checking
- Type hints for all function signatures
- Docstrings for modules, classes, and functions
- Keep functions under 50 lines when possible

**Example**:
```python
from typing import Optional

def parse_job_html(html: str, url: str) -> Optional[dict]:
    """
    Parse job HTML into structured data.
    
    Args:
        html: Raw HTML content
        url: Source URL of the job posting
        
    Returns:
        Parsed job data dictionary or None if parsing fails
    """
    if not html or not url:
        return None
    
    # Implementation
    return parsed_data
```

### General Principles

- **DRY** (Don't Repeat Yourself) - Extract common code into functions
- **SOLID** principles for object-oriented code
- **Clear naming** - Names should be self-documenting
- **Small commits** - One logical change per commit
- **Test coverage** - Aim for 80%+ coverage on new code

---

## Commit Guidelines

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, no logic change)
- **refactor**: Code refactoring
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Build process, tooling, dependencies

### Examples

```
feat(parser): add support for Workday ATS parsing

- Implement Workday-specific HTML selectors
- Add Workday API detection
- Update parser strategy selection logic

Closes #123
```

```
fix(crawler): handle timeout errors in batch crawl

Previously, timeout errors would crash the entire batch.
Now individual timeouts are caught and logged, allowing
other URLs to continue processing.

Fixes #456
```

```
docs(api): update endpoint examples with new parameters

- Add salary_min and salary_max filter examples
- Update response schemas
- Fix typos in query parameter descriptions
```

### Commit Best Practices

- Use imperative mood ("add feature" not "added feature")
- Keep subject line under 72 characters
- Capitalize subject line
- No period at end of subject line
- Separate subject from body with blank line
- Explain what and why, not how

---

## Pull Request Process

### Before Submitting

1. **Update from upstream**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all checks**
   ```bash
   # Linting
   make lint
   
   # Tests
   make test
   
   # Or manually
   cd apps/api && golangci-lint run
   cd apps/parser && uv run ruff check .
   cd apps/parser && uv run mypy .
   ```

3. **Update documentation**
   - Update README if adding features
   - Update API docs if changing endpoints
   - Add inline code comments

4. **Test manually**
   - Start services with your changes
   - Test the actual functionality
   - Check for edge cases

### PR Template

When creating a PR, include:

**Description**
- What does this PR do?
- Why is this change needed?

**Related Issues**
- Fixes #123
- Related to #456

**Type of Change**
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

**Testing**
- How was this tested?
- What test cases were added?

**Checklist**
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Commented complex code
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests passing
- [ ] No new warnings

### Review Process

1. **Automated checks** must pass (linting, tests)
2. **Code review** by maintainers
3. **Discussion** of changes if needed
4. **Approval** from at least one maintainer
5. **Merge** by maintainers

### After Merge

- Delete your branch
- Update your local main branch
- Close any related issues

---

## Reporting Issues

### Before Creating an Issue

- Search existing issues to avoid duplicates
- Test with the latest version
- Gather relevant information (logs, screenshots)

### Bug Reports

Include:

- **Description**: Clear description of the bug
- **Steps to reproduce**: Exact steps to trigger the bug
- **Expected behavior**: What should happen
- **Actual behavior**: What actually happens
- **Environment**: OS, Docker version, Go/Python version
- **Logs**: Relevant error messages or logs
- **Screenshots**: If applicable

**Example**:
```markdown
## Bug: Parser fails on Greenhouse job pages

**Description**
Parser service crashes when processing certain Greenhouse job postings.

**Steps to Reproduce**
1. Start services with `docker-compose up`
2. Trigger workflow for URL: https://boards.greenhouse.io/example/job/123
3. Check parser logs

**Expected Behavior**
Job should be parsed successfully

**Actual Behavior**
Parser crashes with error: "list index out of range"

**Environment**
- OS: macOS 14.0
- Docker: 24.0.6
- Python: 3.12.1

**Logs**
```
[ERROR] Parser failed: list index out of range
Traceback (most recent call last):
  File "main.py", line 123, in parse_job
    title = soup.select('h1')[0]
IndexError: list index out of range
```
```

### Feature Requests

Include:

- **Use case**: What problem does this solve?
- **Description**: Detailed description of the feature
- **Alternatives**: Other solutions you've considered
- **Priority**: How important is this feature?

---

## Project Structure

Understanding the codebase:

```
quiethire/
├── apps/
│   ├── api/                   # Go API Gateway + Worker
│   │   ├── cmd/              # Command-line tools
│   │   │   ├── api/          # REST API server
│   │   │   ├── worker/       # Temporal worker
│   │   │   └── ...           # Utility tools
│   │   └── internal/         # Internal packages
│   │       ├── activities/   # Temporal activities
│   │       └── workflows/    # Temporal workflows
│   ├── crawler-python/       # Python web crawler
│   ├── parser/               # Job HTML parser
│   ├── osint-discovery/      # OSINT discovery service
│   └── proxy-manager/        # Proxy rotation service
├── config/                   # Configuration files
│   ├── clickhouse/          # ClickHouse schemas
│   └── postgres/            # PostgreSQL schemas
├── docs/                     # Documentation
└── docker-compose.yml        # Service orchestration
```

### Key Components

- **API Gateway** (`apps/api/cmd/api`): REST API endpoints
- **Worker** (`apps/api/cmd/worker`): Temporal workflow execution
- **Activities** (`apps/api/internal/activities`): Reusable workflow tasks
- **Workflows** (`apps/api/internal/workflows`): Business logic orchestration
- **Parser** (`apps/parser`): HTML to structured data conversion
- **OSINT** (`apps/osint-discovery`): Company and URL discovery

---

## Testing

### Running Tests

**Go Tests**:
```bash
cd apps/api
go test ./...
go test -v ./internal/activities
go test -race ./...
```

**Python Tests**:
```bash
cd apps/parser
uv run pytest
uv run pytest -v
uv run pytest --cov=.
```

### Writing Tests

**Go Test Example**:
```go
func TestGetStaleCompanies(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer db.Close()
    
    activities := NewDiscoveryActivities(db)
    
    // Test
    companies, err := activities.GetStaleCompanies(context.Background(), 7)
    
    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, companies)
}
```

**Python Test Example**:
```python
import pytest
from main import parse_job_html

def test_parse_job_html_with_valid_data():
    html = "<html><h1>Senior Engineer</h1></html>"
    url = "https://example.com/job/123"
    
    result = parse_job_html(html, url)
    
    assert result is not None
    assert result["title"] == "Senior Engineer"

def test_parse_job_html_with_invalid_data():
    result = parse_job_html("", "")
    assert result is None
```

### Integration Tests

For workflows and end-to-end testing:

```bash
# Start all services
docker-compose up -d

# Run integration tests
cd apps/api
go test -tags=integration ./...
```

---

## Questions?

- **Documentation**: Check [docs/](../docs/)
- **Issues**: Open an issue on GitHub
- **Discussions**: Start a discussion on GitHub

---

## Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes
- Project documentation

Thank you for contributing to QuietHire!

---

**Maintainer**: Rohit Kumar ([@itisrohit](https://github.com/itisrohit))  
**License**: MIT  
**Last Updated**: December 2024
