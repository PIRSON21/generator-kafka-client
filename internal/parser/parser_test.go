package parser

import (
	"errors"
	"testing"

	"github.com/PIRSON21/generator-kafka-client/internal/model"
)

const testdata = "testdata/"

// TestParse_Valid проверяет корректный разбор валидной схемы.
func TestParse_Valid(t *testing.T) {
	def, err := Parse(testdata + "valid.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(def.Services) != 1 {
		t.Fatalf("len(Services) = %d, want 1", len(def.Services))
	}

	svc := def.Services[0]

	if svc.Name != "BillingService" {
		t.Errorf("Services[0].Name = %q, want %q", svc.Name, "BillingService")
	}

	wantMethods := []model.MethodDef{
		{Name: "AddContract", EventType: "add_contract", ReqType: "AddContractRequest"},
		{Name: "NotifyUser", EventType: "notify_user", ReqType: "NotifyUserRequest"},
		{Name: "Ping", EventType: "ping", ReqType: "PingRequest"},
	}
	if len(svc.Methods) != len(wantMethods) {
		t.Fatalf("len(Methods) = %d, want %d", len(svc.Methods), len(wantMethods))
	}
	for i, want := range wantMethods {
		got := svc.Methods[i]
		if got != want {
			t.Errorf("Methods[%d] = %+v, want %+v", i, got, want)
		}
	}

	wantStructs := []string{"AddContractRequest", "NotifyUserRequest", "PingRequest"}
	if len(def.Structs) != len(wantStructs) {
		t.Fatalf("len(Structs) = %d, want %d", len(def.Structs), len(wantStructs))
	}
	for i, name := range wantStructs {
		if def.Structs[i].Name != name {
			t.Errorf("Structs[%d].Name = %q, want %q", i, def.Structs[i].Name, name)
		}
	}
}

// TestParse_FieldTagsGeneratedWhenAbsent проверяет что при отсутствии тегов
// генерируется json:"snake_case".
func TestParse_FieldTagsGeneratedWhenAbsent(t *testing.T) {
	def, err := Parse(testdata + "valid.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(def.Structs) < 1 {
		t.Fatalf("no structs parsed")
	}

	// AddContractRequest: поля без тегов → генерируются json-теги
	addContract := def.Structs[0]
	if len(addContract.Fields) != 2 {
		t.Fatalf("AddContractRequest fields = %d, want 2", len(addContract.Fields))
	}

	wantFields := []model.FieldDef{
		{Name: "ContractID", Type: "string", Tag: `json:"contract_id"`},
		{Name: "Amount", Type: "int", Tag: `json:"amount"`},
	}
	for i, want := range wantFields {
		got := addContract.Fields[i]
		if got != want {
			t.Errorf("Fields[%d] = %+v, want %+v", i, got, want)
		}
	}
}

// TestParse_EmptyStructHasNoFields проверяет что пустая структура корректно разбирается.
func TestParse_EmptyStructHasNoFields(t *testing.T) {
	def, err := Parse(testdata + "valid.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(def.Structs) < 3 {
		t.Fatalf("len(Structs) = %d, want >= 3", len(def.Structs))
	}

	// PingRequest — пустая структура
	ping := def.Structs[2]
	if ping.Name != "PingRequest" {
		t.Fatalf("Structs[2].Name = %q, want %q", ping.Name, "PingRequest")
	}
	if len(ping.Fields) != 0 {
		t.Errorf("PingRequest fields = %d, want 0", len(ping.Fields))
	}
}

// TestParse_FieldTagsCopiedWhenPresent проверяет что существующие теги копируются как есть.
func TestParse_FieldTagsCopiedWhenPresent(t *testing.T) {
	def, err := Parse(testdata + "valid_with_tags.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(def.Structs) < 1 {
		t.Fatalf("no structs parsed")
	}

	// CreateOrderRequest: оба поля имеют теги → копируются как есть
	createOrder := def.Structs[0]
	if createOrder.Name != "CreateOrderRequest" {
		t.Fatalf("Structs[0].Name = %q, want %q", createOrder.Name, "CreateOrderRequest")
	}
	if len(createOrder.Fields) != 2 {
		t.Fatalf("CreateOrderRequest fields = %d, want 2", len(createOrder.Fields))
	}

	wantFields := []model.FieldDef{
		{Name: "OrderID", Type: "string", Tag: `json:"order_id,omitempty"`},
		{Name: "Amount", Type: "int", Tag: `json:"amount"`},
	}
	for i, want := range wantFields {
		got := createOrder.Fields[i]
		if got != want {
			t.Errorf("Fields[%d] = %+v, want %+v", i, got, want)
		}
	}
}

// TestParse_FieldTagMixedCopiedAndGenerated проверяет что в одной структуре
// теги могут быть частично заданы (одни копируются, другие генерируются).
func TestParse_FieldTagMixedCopiedAndGenerated(t *testing.T) {
	def, err := Parse(testdata + "valid_with_tags.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(def.Structs) < 2 {
		t.Fatalf("len(Structs) = %d, want >= 2", len(def.Structs))
	}

	// CancelOrderRequest: поле OrderID без тега → генерируется json:"order_id"
	cancelOrder := def.Structs[1]
	if cancelOrder.Name != "CancelOrderRequest" {
		t.Fatalf("Structs[1].Name = %q, want %q", cancelOrder.Name, "CancelOrderRequest")
	}
	if len(cancelOrder.Fields) != 1 {
		t.Fatalf("CancelOrderRequest fields = %d, want 1", len(cancelOrder.Fields))
	}

	got := cancelOrder.Fields[0]
	want := model.FieldDef{Name: "OrderID", Type: "string", Tag: `json:"order_id"`}
	if got != want {
		t.Errorf("Fields[0] = %+v, want %+v", got, want)
	}
}

// TestParse_TwoInterfaces проверяет корректный разбор схемы с двумя интерфейсами.
func TestParse_TwoInterfaces(t *testing.T) {
	def, err := Parse(testdata + "two_interfaces.go")
	if err != nil {
		t.Fatalf("unexpected error for two interfaces: %v", err)
	}

	if len(def.Services) != 2 {
		t.Fatalf("len(Services) = %d, want 2", len(def.Services))
	}

	wantNames := []string{"FirstService", "SecondService"}
	for i, want := range wantNames {
		if def.Services[i].Name != want {
			t.Errorf("Services[%d].Name = %q, want %q", i, def.Services[i].Name, want)
		}
	}

	if len(def.Structs) != 2 {
		t.Fatalf("len(Structs) = %d, want 2", len(def.Structs))
	}
}

// TestParse_WrongPackage проверяет ошибку при неверном имени пакета.
func TestParse_WrongPackage(t *testing.T) {
	_, err := Parse(testdata + "wrong_package.go")
	if err == nil {
		t.Fatal("expected error for wrong package name, got nil")
	}
}

// TestParse_NoInterface проверяет ошибку при отсутствии интерфейса.
func TestParse_NoInterface(t *testing.T) {
	_, err := Parse(testdata + "no_interface.go")
	if err == nil {
		t.Fatal("expected error for no interface, got nil")
	}
}

// TestParse_WrongSignatureNoCtx проверяет ошибку при отсутствии context.Context.
func TestParse_WrongSignatureNoCtx(t *testing.T) {
	_, err := Parse(testdata + "wrong_signature_no_ctx.go")
	if err == nil {
		t.Fatal("expected error for missing context.Context, got nil")
	}
}

// TestParse_WrongSignatureNoError проверяет ошибку при отсутствии возвращаемого error.
func TestParse_WrongSignatureNoError(t *testing.T) {
	_, err := Parse(testdata + "wrong_signature_no_error.go")
	if err == nil {
		t.Fatal("expected error for missing error return, got nil")
	}
}

// TestParse_MissingStruct проверяет ошибку когда структура-аргумент не объявлена.
func TestParse_MissingStruct(t *testing.T) {
	_, err := Parse(testdata + "missing_struct.go")
	if err == nil {
		t.Fatal("expected error for missing request struct, got nil")
	}
}

// TestParse_FileNotFound проверяет ошибку при несуществующем файле.
func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse(testdata + "nonexistent.go")
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}
