# monkeypatch-go

MonkeyPatch-Go is a lightweight Go library that allows runtime function patching, commonly known as monkey patching. This enables replacing functions or instance methods dynamically, making it useful for testing, debugging, and altering behavior at runtime.

## Features

- Patch global functions
- Patch instance methods
- Restore patched functions
- Unpatch all functions
- Safe patching with reversion support

## Warning

Monkey patching relies on unsafe operations and may break across different Go versions due to compiler optimizations. Use with caution.

## Installation

```bash
go get github.com/TFMV/monkeypatch-go
```

## Usage

### 1️⃣ Patching a Global Function

```go
package main

import (
    "fmt"
    "github.com/yourusername/monkeypatch-go"
)

// Original function
func Greet() string {
    return "Hello, World!"
}

// Replacement function
func FakeGreet() string {
    return "You've been monkey patched!"
}

func main() {
    fmt.Println(Greet()) // Output: Hello, World!

    guard, err := monkey.Patch(Greet, FakeGreet)
    if err != nil {
        panic(err)
    }
    fmt.Println(Greet()) // Output: You've been monkey patched!

    guard.Unpatch() // Restore original
    fmt.Println(Greet()) // Output: Hello, World!
}
```

### 2️⃣ Patching an Instance Method

```go
package main

import (
    "fmt"
    "reflect"
    "github.com/yourusername/monkeypatch-go"
)

type Person struct{}

// Original method
func (p *Person) SayHello() string {
    return "Hello from Person"
}

// Replacement method
func FakeSayHello(_ *Person) string {
    return "Patched Hello!"
}

func main() {
    p := &Person{}
    fmt.Println(p.SayHello()) // Output: Hello from Person

    guard, err := monkey.PatchInstanceMethod(reflect.TypeOf(p), "SayHello", FakeSayHello)
    if err != nil {
        panic(err)
    }
    fmt.Println(p.SayHello()) // Output: Patched Hello!

    guard.Unpatch() // Restore original
    fmt.Println(p.SayHello()) // Output: Hello from Person
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
