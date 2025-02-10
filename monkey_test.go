package monkey_test

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"unsafe"

	monkey "github.com/TFMV/monkeypatch-go"
)

var (
	mu           sync.Mutex
	patchedFuncs = make(map[uintptr]patchRecord)
)

// patchRecord holds the original bytes of a patched function and the replacement value.
type patchRecord struct {
	originalBytes []byte
	replacement   reflect.Value
}

// getPtr retrieves the underlying function pointer from a reflect.Value.
func getPtr(v reflect.Value) unsafe.Pointer {
	return (*struct {
		_   uintptr
		ptr unsafe.Pointer
	})(unsafe.Pointer(&v)).ptr
}

// PatchGuard holds information about an applied patch.
type PatchGuard struct {
	target      reflect.Value
	replacement reflect.Value
}

// Unpatch removes the applied patch from the target function.
func (g *PatchGuard) Unpatch() {
	unpatchValue(g.target)
}

// Restore re-applies the patch to the target function.
func (g *PatchGuard) Restore() {
	patchValue(g.target, g.replacement)
}

// Patch replaces a function implementation with another.
func Patch(target, replacement interface{}) (*PatchGuard, error) {
	t := reflect.ValueOf(target)
	r := reflect.ValueOf(replacement)

	if err := validateFuncPair(t, r); err != nil {
		return nil, err
	}

	patchValue(t, r)
	return &PatchGuard{
		target:      t,
		replacement: r,
	}, nil
}

// PatchInstanceMethod replaces a method of a type.
func PatchInstanceMethod(target reflect.Type, methodName string, replacement interface{}) (*PatchGuard, error) {
	method, ok := target.MethodByName(methodName)
	if !ok {
		return nil, fmt.Errorf("unknown method %s", methodName)
	}
	r := reflect.ValueOf(replacement)

	if err := validateFuncPair(method.Func, r); err != nil {
		return nil, err
	}

	patchValue(method.Func, r)
	return &PatchGuard{
		target:      method.Func,
		replacement: r,
	}, nil
}

// patchValue applies the patch.
func patchValue(target, replacement reflect.Value) {
	mu.Lock()
	defer mu.Unlock()

	targetPtr := target.Pointer()
	if rec, ok := patchedFuncs[targetPtr]; ok {
		// If already patched with the same function, do nothing.
		if reflect.DeepEqual(rec.replacement.Interface(), replacement.Interface()) {
			return
		}
		unpatch(targetPtr, rec)
	}

	originalBytes := replaceFunction(targetPtr, uintptr(getPtr(replacement)))
	patchedFuncs[targetPtr] = patchRecord{
		originalBytes: originalBytes,
		replacement:   replacement,
	}
}

// Unpatch removes a patch from a function.
func Unpatch(target interface{}) bool {
	return unpatchValue(reflect.ValueOf(target))
}

// UnpatchInstanceMethod removes a patch from a method.
func UnpatchInstanceMethod(target reflect.Type, methodName string) bool {
	method, ok := target.MethodByName(methodName)
	if !ok {
		panic(fmt.Sprintf("unknown method %s", methodName))
	}
	return unpatchValue(method.Func)
}

// UnpatchAll removes all applied patches.
func UnpatchAll() {
	mu.Lock()
	defer mu.Unlock()
	for ptr, rec := range patchedFuncs {
		unpatch(ptr, rec)
		delete(patchedFuncs, ptr)
	}
}

// unpatchValue removes a patch if it exists.
func unpatchValue(target reflect.Value) bool {
	mu.Lock()
	defer mu.Unlock()

	targetPtr := target.Pointer()
	rec, ok := patchedFuncs[targetPtr]
	if !ok {
		return false
	}
	unpatch(targetPtr, rec)
	delete(patchedFuncs, targetPtr)
	return true
}

// unpatch restores the original function bytes.
func unpatch(targetPtr uintptr, rec patchRecord) {
	copyToLocation(targetPtr, rec.originalBytes)
}

// validateFuncPair ensures target and replacement are valid functions.
func validateFuncPair(target, replacement reflect.Value) error {
	if target.Kind() != reflect.Func {
		return fmt.Errorf("target must be a function")
	}
	if replacement.Kind() != reflect.Func {
		return fmt.Errorf("replacement must be a function")
	}
	if target.Type() != replacement.Type() {
		return fmt.Errorf("target and replacement must have the same type: %s != %s", target.Type(), replacement.Type())
	}
	return nil
}

// replaceFunction replaces the function in memory.
func replaceFunction(targetPtr uintptr, newPtr uintptr) []byte {
	oldBytes := make([]byte, 100)
	copy(oldBytes, (*[100]byte)(unsafe.Pointer(targetPtr))[:])

	copy((*[100]byte)(unsafe.Pointer(targetPtr))[:], (*[100]byte)(unsafe.Pointer(newPtr))[:])

	return oldBytes
}

// copyToLocation copies the given bytes to restore the function.
func copyToLocation(targetPtr uintptr, data []byte) {
	copy((*[100]byte)(unsafe.Pointer(targetPtr))[:], data)
}

// Original function
func sayHello() string {
	return "Hello, World!"
}

// Replacement function
func sayGoodbye() string {
	return "Goodbye, World!"
}

// Define a struct with a method
type Greeter struct{}

func (g Greeter) Greet() string {
	return "Hello"
}

// Replacement method
func (g Greeter) GreetReplacement() string {
	return "Hi"
}

func TestPatchFunction(t *testing.T) {
	// Ensure original function works
	original := sayHello()
	if original != "Hello, World!" {
		t.Fatalf("Expected 'Hello, World!', got '%s'", original)
	}

	// Patch the sayHello function
	guard, err := monkey.Patch(sayHello, sayGoodbye)
	if err != nil {
		t.Fatalf("Failed to patch function: %v", err)
	}
	defer guard.Unpatch()

	// Test that sayHello now returns "Goodbye, World!"
	patched := sayHello()
	if patched != "Goodbye, World!" {
		t.Fatalf("Expected 'Goodbye, World!', got '%s'", patched)
	}
}

func TestUnpatch(t *testing.T) {
	// Patch the sayHello function
	guard, err := monkey.Patch(sayHello, sayGoodbye)
	if err != nil {
		t.Fatalf("Failed to patch function: %v", err)
	}

	// Unpatch the function
	guard.Unpatch()

	// Test that sayHello returns the original value
	result := sayHello()
	if result != "Hello, World!" {
		t.Fatalf("Expected 'Hello, World!', got '%s'", result)
	}
}

func TestPatchInstanceMethod(t *testing.T) {
	var greeter Greeter

	// Ensure original method works
	original := greeter.Greet()
	if original != "Hello" {
		t.Fatalf("Expected 'Hello', got '%s'", original)
	}

	// Patch the Greet method
	guard, err := monkey.PatchInstanceMethod(reflect.TypeOf(greeter), "Greet", Greeter.GreetReplacement)
	if err != nil {
		t.Fatalf("Failed to patch instance method: %v", err)
	}
	defer guard.Unpatch()

	// Test that Greet now returns "Hi"
	patched := greeter.Greet()
	if patched != "Hi" {
		t.Fatalf("Expected 'Hi', got '%s'", patched)
	}
}

func TestUnpatchInstanceMethod(t *testing.T) {
	var greeter Greeter

	// Patch the Greet method
	guard, err := monkey.PatchInstanceMethod(reflect.TypeOf(greeter), "Greet", Greeter.GreetReplacement)
	if err != nil {
		t.Fatalf("Failed to patch instance method: %v", err)
	}

	// Unpatch the method
	guard.Unpatch()

	// Test that Greet returns the original value
	result := greeter.Greet()
	if result != "Hello" {
		t.Fatalf("Expected 'Hello', got '%s'", result)
	}
}

func TestUnpatchAll(t *testing.T) {
	// Patch multiple functions
	guard1, err := monkey.Patch(sayHello, sayGoodbye)
	if err != nil {
		t.Fatalf("Failed to patch sayHello: %v", err)
	}

	var greeter Greeter
	guard2, err := monkey.PatchInstanceMethod(reflect.TypeOf(greeter), "Greet", Greeter.GreetReplacement)
	if err != nil {
		t.Fatalf("Failed to patch Greeter.Greet: %v", err)
	}

	// Unpatch all
	monkey.UnpatchAll()

	// Test that sayHello returns the original value
	result := sayHello()
	if result != "Hello, World!" {
		t.Fatalf("Expected 'Hello, World!', got '%s'", result)
	}

	// Test that Greeter.Greet returns the original value
	greetResult := greeter.Greet()
	if greetResult != "Hello" {
		t.Fatalf("Expected 'Hello', got '%s'", greetResult)
	}

	// Clean up guards
	guard1.Unpatch()
	guard2.Unpatch()
}
