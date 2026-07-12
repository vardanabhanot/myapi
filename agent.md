# myAPI Agent Reference

## Project Overview
myAPI is a simple API testing tool built in Go using the Fyne GUI toolkit. It serves as a lightweight, native alternative to tools like Postman or Insomnia, allowing users to construct HTTP requests, view responses, manage history, and generate code snippets for different programming languages.

## Tech Stack
- **Language:** Go
- **GUI Framework:** Fyne (v2)
- **Architecture:** Native desktop executable with modular internal packages (`core` and `ui`).

## Directory Structure
- **`core/`**: Contains the core business logic of the application.
  - `request.go`: Handles the construction and execution of HTTP requests.
  - `history.go`: Manages request history persistence and retrieval.
  - `collection.go`: Data structures and logic for organizing endpoints into collections.
  - **`codegen/`**: Code generator logic to convert HTTP requests into code snippets (e.g., `curl.go`, `php.go`, `go.go`).
- **`ui/`**: Contains the Fyne GUI implementation.
  - `gui.go`: Main entry point for constructing the primary UI layout and wiring up components.
  - `request.go`, `response.go`, `responseContainer.go`: Complex UI components for the request builder and response viewer.
  - `baseTheme.go`, `footerTheme.go`, `overridePaddingTheme.go`: Custom Fyne theme definitions to style the app.
  - `hoverableListItem.go`, `tappableIcon.go`, `split.go`: Custom extended Fyne widgets for enhanced interactions.
- **`assets/`**: Static assets (icons, fonts) used by the application.
- **`main.go`**: Application entry point. Initializes the Fyne app, sets the custom theme, and boots up the main window.
- **`FyneApp.toml`**: Metadata configuration for the Fyne application packaging.

## Key Concepts and Patterns
1. **Fyne Toolkit Usage:**
   - The UI is entirely built using Fyne's canvas objects, widgets, and layout containers.
   - Heavy use of custom widgets extending standard Fyne widgets to add interactive behaviors like hovering and custom tapping.
   - Theming is deeply customized by implementing and extending `fyne.Theme` interfaces.
2. **Separation of Concerns:**
   - **`core`**: Pure Go logic, independent of UI components. Handles networking, file I/O (history), and code generation.
   - **`ui`**: Reacts to user input and calls functions from the `core` package, managing visual state.
3. **Code Generation (`core/codegen/`):**
   - Implements a generic generator pattern that makes it easy to add new export formats by defining new language-specific generators.

## Current Status & Known Issues (TODO)
The project is functional but in active development.

**Completed Features:**
- HTTP request execution (with cancellation).
- History tracking.
- Code generator baseline (with basic cURL and PHP support).

**Upcoming Features & TODOs:**
- Workspace and Environment Variables support.
- Collections to group similar API endpoints.
- Support for multipart forms.
- Syntax highlighting for Response views.
- UI improvements (Spinner for active requests, shortcut to hide response container).

**Known Issues:**
- **Performance/Memory Leak:** There is a known memory issue that occurs when many tabs are used with content exceeding 200 characters, causing memory usage to increase exponentially. Be cautious when handling large string buffers in the UI.

## Agent Development Guidelines
- **UI Modifications:** Always adhere to Fyne v2 framework paradigms. Utilize Fyne layouts (`container.New...`) and standard widgets. Avoid trying to use web paradigms (HTML/CSS/JS) as this is a purely native application.
- **Adding Generators:** When adding new code generator targets, add them to `core/codegen/` and ensure they follow the existing interface/structure established by `curl.go` and `php.go`.
- **Handling Data:** Keep UI responsive. If performing heavy I/O or network requests, ensure they run in goroutines and do not block the main Fyne UI thread. Use Fyne's data binding or callback mechanisms to update the UI thread safely.
