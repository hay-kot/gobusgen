package generator_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hay-kot/gobusgen/internal/generator"
	"github.com/hay-kot/gobusgen/internal/parser"
)

// TestIntegration_ParseAndGenerate exercises the full pipeline: write Go source
// with event types and a map variable, parse it, generate the event bus, write
// both files into a temp directory, and verify the result compiles.
func TestIntegration_ParseAndGenerate(t *testing.T) {
	dir := t.TempDir()

	source := `package demo

type OrderCreated struct {
	OrderID string
	Amount  float64
}

type OrderShipped struct {
	OrderID    string
	TrackingNo string
}

var Events = map[string]any{
	"order.created": OrderCreated{},
	"order.shipped": OrderShipped{},
}
`
	if err := os.WriteFile(filepath.Join(dir, "events.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	input, err := parser.Parse(dir, "Events")
	if err != nil {
		t.Fatalf("parser.Parse: %v", err)
	}

	if input.PackageName != "demo" {
		t.Fatalf("PackageName = %q, want %q", input.PackageName, "demo")
	}

	if len(input.Events) != 2 {
		t.Fatalf("Events count = %d, want 2", len(input.Events))
	}

	src, err := generator.Generate(input)
	if err != nil {
		t.Fatalf("generator.Generate: %v", err)
	}

	outPath := filepath.Join(dir, "eventbus.gen.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a go.mod so `go vet` works in the temp directory
	goMod := "module demo\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify the generated code compiles
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code does not compile:\n%s\n%v", out, err)
	}
}

// TestIntegration_MultiPackage verifies the parse+generate pipeline works for
// two independent packages in the same module with different prefixes.
func TestIntegration_MultiPackage(t *testing.T) {
	root := t.TempDir()
	pkgA := filepath.Join(root, "pkga")
	pkgB := filepath.Join(root, "pkgb")
	if err := os.MkdirAll(pkgA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0o755); err != nil {
		t.Fatal(err)
	}

	// pkgA: var Events → empty prefix (backward compatible)
	srcA := `package pkga

type UserCreated struct {
	UserID string
}

var Events = map[string]any{
	"user.created": UserCreated{},
}
`
	// pkgB: var Commands → "Command" prefix (derived)
	srcB := `package pkgb

type PlaceOrder struct {
	OrderID string
}

type CancelOrder struct {
	OrderID string
}

var Commands = map[string]any{
	"order.place":  PlaceOrder{},
	"order.cancel": CancelOrder{},
}
`
	if err := os.WriteFile(filepath.Join(pkgA, "events.go"), []byte(srcA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgB, "commands.go"), []byte(srcB), 0o644); err != nil {
		t.Fatal(err)
	}

	goMod := "module multitest\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Generate for package A (empty prefix)
	inputA, err := parser.Parse(pkgA, "Events")
	if err != nil {
		t.Fatalf("parser.Parse pkgA: %v", err)
	}
	if inputA.Prefix != "" {
		t.Fatalf("pkgA prefix = %q, want empty", inputA.Prefix)
	}

	genA, err := generator.Generate(inputA)
	if err != nil {
		t.Fatalf("generator.Generate pkgA: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgA, "eventbus.gen.go"), genA, 0o644); err != nil {
		t.Fatal(err)
	}

	// Generate for package B ("Command" prefix)
	inputB, err := parser.Parse(pkgB, "Commands")
	if err != nil {
		t.Fatalf("parser.Parse pkgB: %v", err)
	}
	if inputB.Prefix != "Commands" {
		t.Fatalf("pkgB prefix = %q, want %q", inputB.Prefix, "Commands")
	}

	genB, err := generator.Generate(inputB)
	if err != nil {
		t.Fatalf("generator.Generate pkgB: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgB, "commandsbus.gen.go"), genB, 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify both packages compile
	vetCmd := exec.Command("go", "vet", "./...")
	vetCmd.Dir = root
	vetOut, vetErr := vetCmd.CombinedOutput()
	if vetErr != nil {
		t.Fatalf("generated code does not compile:\n%s\n%v", vetOut, vetErr)
	}
}

// TestIntegration_RuntimeBehavior generates an event bus into a temp package,
// writes a test file that exercises hooks, non-blocking publish, and panic
// recovery at runtime, then runs go test in that directory.
func TestIntegration_RuntimeBehavior(t *testing.T) {
	dir := t.TempDir()

	source := `package demo

type OrderCreated struct {
	OrderID string
}

type OrderShipped struct {
	OrderID    string
	TrackingNo string
}

var Events = map[string]any{
	"order.created": OrderCreated{},
	"order.shipped": OrderShipped{},
}
`
	if err := os.WriteFile(filepath.Join(dir, "events.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	input, err := parser.Parse(dir, "Events")
	if err != nil {
		t.Fatalf("parser.Parse: %v", err)
	}

	src, err := generator.Generate(input)
	if err != nil {
		t.Fatalf("generator.Generate: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "eventbus.gen.go"), src, 0o644); err != nil {
		t.Fatal(err)
	}

	goMod := "module demo\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	testFile := `package demo

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestOnPublishHook(t *testing.T) {
	bus := New(10)

	var count atomic.Int32
	bus.OnPublish(func(e Event, p any) {
		count.Add(1)
	})

	bus.PublishOrderCreated(OrderCreated{OrderID: "1"})
	bus.PublishOrderShipped(OrderShipped{OrderID: "2", TrackingNo: "T1"})

	if got := count.Load(); got != 2 {
		t.Errorf("OnPublish called %d times, want 2", got)
	}
}

func TestOnDropHook(t *testing.T) {
	bus := New(1) // buffer of 1

	var published, dropped atomic.Int32
	bus.OnPublish(func(e Event, p any) { published.Add(1) })
	bus.OnDrop(func(e Event, p any) { dropped.Add(1) })

	// First publish fills the buffer
	bus.PublishOrderCreated(OrderCreated{OrderID: "1"})
	// Second publish should drop (nobody is consuming)
	bus.PublishOrderCreated(OrderCreated{OrderID: "2"})

	if got := published.Load(); got != 1 {
		t.Errorf("OnPublish called %d times, want 1", got)
	}
	if got := dropped.Load(); got != 1 {
		t.Errorf("OnDrop called %d times, want 1", got)
	}
}

func TestOnSubscribeHook(t *testing.T) {
	bus := New(10)

	var events []Event
	bus.OnSubscribe(func(e Event) {
		events = append(events, e)
	})

	bus.SubscribeOrderCreated(func(OrderCreated) {})
	bus.SubscribeOrderShipped(func(OrderShipped) {})

	if len(events) != 2 {
		t.Fatalf("OnSubscribe called %d times, want 2", len(events))
	}
	if events[0] != EventOrderCreated {
		t.Errorf("first event = %q, want %q", events[0], EventOrderCreated)
	}
	if events[1] != EventOrderShipped {
		t.Errorf("second event = %q, want %q", events[1], EventOrderShipped)
	}
}

func TestPanicRecovery(t *testing.T) {
	bus := New(10)

	received := make(chan string, 1)

	bus.SubscribeOrderCreated(func(e OrderCreated) {
		if e.OrderID == "panic" {
			panic("boom")
		}
		received <- e.OrderID
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Start(ctx)

	// First event panics, event loop should survive
	bus.PublishOrderCreated(OrderCreated{OrderID: "panic"})
	// Second event should be processed
	bus.PublishOrderCreated(OrderCreated{OrderID: "ok"})

	select {
	case id := <-received:
		if id != "ok" {
			t.Errorf("received OrderID = %q, want %q", id, "ok")
		}
	case <-time.After(time.Second):
		t.Fatal("event loop stopped after subscriber panic")
	}
}

func TestOnPanicHook(t *testing.T) {
	bus := New(10)

	panicInfo := make(chan any, 1)
	bus.OnPanic(func(e Event, p any, r any) {
		panicInfo <- r
	})

	bus.SubscribeOrderCreated(func(e OrderCreated) {
		panic("subscriber boom")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Start(ctx)

	bus.PublishOrderCreated(OrderCreated{OrderID: "1"})

	select {
	case r := <-panicInfo:
		if r != "subscriber boom" {
			t.Errorf("panic value = %v, want %q", r, "subscriber boom")
		}
	case <-time.After(time.Second):
		t.Fatal("OnPanic hook not called")
	}
}

func TestOnPanicHookPanicDoesNotCrashLoop(t *testing.T) {
	bus := New(10)

	// OnPanic hook that itself panics
	bus.OnPanic(func(e Event, p any, r any) {
		panic("hook panic")
	})

	received := make(chan string, 1)
	bus.SubscribeOrderCreated(func(e OrderCreated) {
		if e.OrderID == "panic" {
			panic("boom")
		}
		received <- e.OrderID
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Start(ctx)

	// Triggers subscriber panic → OnPanic hook panic → should be swallowed
	bus.PublishOrderCreated(OrderCreated{OrderID: "panic"})
	bus.PublishOrderCreated(OrderCreated{OrderID: "ok"})

	select {
	case id := <-received:
		if id != "ok" {
			t.Errorf("received OrderID = %q, want %q", id, "ok")
		}
	case <-time.After(time.Second):
		t.Fatal("event loop stopped after OnPanic hook panic")
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "eventbus_test.go"), []byte(testFile), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "-v", "-count=1", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("runtime tests failed:\n%s\n%v", out, err)
	}
	t.Logf("runtime test output:\n%s", out)
}

func TestIntegration_ConstKeys(t *testing.T) {
	dir := t.TempDir()

	consts := `package demo

type EventName string

const (
	OrderCreatedEvent EventName = "order.created"
	OrderShippedEvent EventName = "order.shipped"
)
`

	source := `package demo

type OrderCreated struct {
	OrderID string
	Amount  float64
}

type OrderShipped struct {
	OrderID    string
	TrackingNo string
}

var Events = map[string]any{
	string(OrderCreatedEvent): OrderCreated{},
	string(OrderShippedEvent): OrderShipped{},
}
`
	if err := os.WriteFile(filepath.Join(dir, "consts.go"), []byte(consts), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	input, err := parser.Parse(dir, "Events")
	if err != nil {
		t.Fatalf("parser.Parse: %v", err)
	}

	if len(input.Events) != 2 {
		t.Fatalf("Events count = %d, want 2", len(input.Events))
	}

	src, err := generator.Generate(input)
	if err != nil {
		t.Fatalf("generator.Generate: %v", err)
	}

	outPath := filepath.Join(dir, "eventbus.gen.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		t.Fatal(err)
	}

	goMod := "module demo\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code does not compile:\n%s\n%v", out, err)
	}
}
