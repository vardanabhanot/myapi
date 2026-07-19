<img src="/myapi-logo-light.png" alt="myAPI logo" title="myAPI" align="left" height="60px"/>
<br>

# myAPI

**The API client that's just an app.** myAPI is a fast, native API testing tool written in Go with the [Fyne](https://fyne.io) toolkit. No account, no cloud, no Electron — open the app, send the request.

**Website:** [vardana.dev/myapi](https://vardana.dev/myapi/) · **Download:** [latest release](https://github.com/vardanabhanot/myapi/releases/latest)

> [!NOTE]
> myAPI is in **alpha**. It's usable for day-to-day request testing, but expect rough edges and breaking changes between releases. Bug reports are very welcome.

## Screenshots

![myAPI in light mode](https://github.com/vardanabhanot/myapi/blob/main/my-api-dev-state.png)
![myAPI in dark mode](https://github.com/vardanabhanot/myapi/blob/main/my-api-dev-state-dark.png)

## Features

- **Native & offline** — one small binary, no browser runtime; your data never leaves your disk
- **Collections** — group related endpoints, rename and reorganize them as your API grows
- **Request history** — every request you send is saved locally
- **Tabs** — work on several requests side by side
- **Environment variables** — define `{{variables}}` once, reuse them across URLs, headers, and bodies; quick-switch environments from the footer
- **Auth** — API Key and OAuth 2.0
- **cURL import & export** — paste a cURL command to create a request, or copy any request out as cURL
- **Code generation** — turn any request into cURL, Go, JavaScript, PHP, or Python code
- **Syntax-highlighted responses**, request timing, and cancellable in-flight requests
- **Light & dark themes**

## Install

Grab a build from the [releases page](https://github.com/vardanabhanot/myapi/releases/latest):

| Platform | Asset |
|----------|-------|
| Windows  | `myapi-windows-amd64.zip` |
| Linux    | `myapi-linux-amd64.tar.xz` |
| macOS    | coming soon |

Unpack and run — there's no installer and nothing else to set up.

### Build from source

Requires [Go](https://go.dev) and the [Fyne prerequisites](https://docs.fyne.io/started/) (a C compiler; on Linux also the graphics dev packages):

```sh
git clone https://github.com/vardanabhanot/myapi
cd myapi
go build .
```

## Roadmap

- [ ] macOS builds
- [ ] Workspaces
- [ ] OpenAPI / Postman import
- [ ] Endpoint documentation
- [ ] Multipart form support
- [ ] Request spinner while a request is in flight

## Contributing

myAPI is built in the open. [Star the repo](https://github.com/vardanabhanot/myapi), [file an issue](https://github.com/vardanabhanot/myapi/issues), or send a pull request.

## License

[MIT](LICENSE)
