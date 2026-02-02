# Coding Guidelines

## Global Variables

This collection uses global variables that can be shared across all roles, see the documnetation on [global variables](./GLOBAL.md).

## Feature Flags Pattern

This collection uses a "_with_" naming convention for optional feature flags in roles. These boolean variables enable or disable specific functionality within a role:

- Feature flags follow the pattern: `<role_name>_with_<feature>`
- Examples: `users_with_sudo`, `traefik_with_acme`
- This pattern allows roles to have a core functionality with optional extensions
- Feature flags should default to `false` to maintain backward compatibility

## Role Development Guidelines

### Directory Structure

Each role should follow this structure:
```
roles/
  role_name/
    README.md         # Required for Galaxy
    meta/main.yml     # Required for Galaxy
    defaults/main.yml # Default variables
    tasks/main.yml    # Main task file
    handlers/main.yml # Handler definitions
    templates/        # Jinja2 templates
    files/           # Static files
    vars/            # Variables
```

### Task Organization

- Use descriptive task names that explain what the task does
- Group related tasks in separate files and include them from main.yml
- Use tags for optional functionality
- Always use fully qualified collection names (FQCN) for modules

### Variable Naming

- Role-specific variables should be prefixed with the role name
- Use underscores to separate words in variable names
- Document all variables in the role's README.md

### Error Handling

- Use `failed_when` and `changed_when` appropriately
- Provide meaningful error messages
- Use `block`/`rescue` for complex error handling scenarios

## Testing

- Test roles on all supported platforms (Debian bookworm/bullseye, Ubuntu jammy/focal)
- Verify idempotency by running roles multiple times
- Check for proper cleanup in handlers

## Documentation

- Every role must have a comprehensive README.md
- Document all variables with their types, defaults, and descriptions
- Provide usage examples in the README
- Keep CHANGELOG.md updated with all changes
