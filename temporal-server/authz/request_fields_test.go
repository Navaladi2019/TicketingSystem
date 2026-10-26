package authz

// Unit tests of the CI-C drift golden's machinery (TestRequestFieldGolden in
// policy_test.go) against synthetic descriptors, so each failure mode is shown
// without waiting for a go.temporal.io/api bump.

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func scalarField(name string, number int32, typ descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(number),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:   typ.Enum(),
	}
}

func messageField(name string, number int32, typeName string) *descriptorpb.FieldDescriptorProto {
	f := scalarField(name, number, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	f.TypeName = proto.String(".synthetic.v1." + typeName)
	return f
}

// syntheticRequest builds `synthetic.v1.Request` from its fields, next to a
// recursive Failure (Failure.cause is a Failure, and Failure.info.back leads
// back to it through Info).
func syntheticRequest(t *testing.T, fields ...*descriptorpb.FieldDescriptorProto) protoreflect.MessageDescriptor {
	t.Helper()
	str := descriptorpb.FieldDescriptorProto_TYPE_STRING
	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("synthetic/v1/request.proto"),
		Package: proto.String("synthetic.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Request"), Field: fields},
			{Name: proto.String("Failure"), Field: []*descriptorpb.FieldDescriptorProto{
				scalarField("message", 1, str),
				messageField("cause", 2, "Failure"),
				messageField("info", 3, "Info"),
			}},
			{Name: proto.String("Info"), Field: []*descriptorpb.FieldDescriptorProto{
				scalarField("detail", 1, str),
				messageField("back", 2, "Failure"),
			}},
		},
	}
	fd, err := protodesc.NewFile(file, nil)
	if err != nil {
		t.Fatalf("synthetic descriptor: %v", err)
	}
	return fd.Messages().ByName("Request")
}

// baseFields is a request with no id- or queue-shaped field: namespace, a
// repeated string, and a recursive Failure.
func baseFields() []*descriptorpb.FieldDescriptorProto {
	str := descriptorpb.FieldDescriptorProto_TYPE_STRING
	tags := scalarField("tags", 2, str)
	tags.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	return []*descriptorpb.FieldDescriptorProto{
		scalarField("namespace", 1, str),
		tags,
		messageField("failure", 3, "Failure"),
	}
}

func renderSynthetic(t *testing.T, method string, ids []Identity, fields ...*descriptorpb.FieldDescriptorProto) (string, []string) {
	t.Helper()
	return renderRequestFieldGolden([]allowedRequest{{method: method, ids: ids, input: syntheticRequest(t, fields...)}})
}

func TestRequestFieldGoldenFailsOnDrift(t *testing.T) {
	want, unclassified := renderSynthetic(t, "Synthetic", []Identity{Relay}, baseFields()...)
	if len(unclassified) != 0 {
		t.Fatalf("base request has unclassified fields: %v", unclassified)
	}
	if again, _ := renderSynthetic(t, "Synthetic", []Identity{Relay}, baseFields()...); goldenMismatch(want, again) != nil {
		t.Fatal("the same descriptor rendered twice gives different goldens")
	}

	added := append(baseFields(), scalarField("note", 4, descriptorpb.FieldDescriptorProto_TYPE_STRING))
	removed := baseFields()[:2]
	renamed := baseFields()
	renamed[1].Name = proto.String("labels") // same number, new name
	retyped := baseFields()
	retyped[1].Type = descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()

	for name, fields := range map[string][]*descriptorpb.FieldDescriptorProto{
		"added": added, "removed": removed, "renamed": renamed, "retyped": retyped,
	} {
		got, _ := renderSynthetic(t, "Synthetic", []Identity{Relay}, fields...)
		if problems := requestFieldProblems(want, got, nil); len(problems) == 0 {
			t.Errorf("%s field: golden still matches", name)
		} else {
			t.Logf("%s field: %s", name, problems[0])
		}
	}
}

func TestRequestFieldGoldenFailsOnUnclassifiedIDField(t *testing.T) {
	str := descriptorpb.FieldDescriptorProto_TYPE_STRING
	for _, name := range []string{"workflow_id", "parent_workflow_id", "task_queue", "sticky_task_queue", "workflow_type", "task_token", "normal_name"} {
		fields := append(baseFields(), scalarField(name, 9, str))
		got, unclassified := renderSynthetic(t, "Synthetic", []Identity{Worker}, fields...)
		// The golden matches (want == got), and the field still fails.
		problems := requestFieldProblems(got, got, unclassified)
		if len(problems) != 1 || !strings.Contains(problems[0], "Synthetic "+name+" for [nava-worker]") {
			t.Errorf("%s: problems = %v, want one unclassified-field failure", name, problems)
		}
		if !strings.Contains(got, "| `"+name+"` | `string` | **unclassified** |") {
			t.Errorf("%s: golden row does not say unclassified:\n%s", name, got)
		}
	}

	// A classified id field passes, and the same field is still unclassified for
	// a second identity with no rule for it.
	fields := append(baseFields(), scalarField("sticky_task_queue", 9, str))
	got, unclassified := renderSynthetic(t, "ShutdownWorker", []Identity{Worker}, fields...)
	if len(unclassified) != 0 || !strings.Contains(got, "| `sticky_task_queue` | `string` | "+classD9gSticky+" |") {
		t.Errorf("classified field: unclassified=%v golden:\n%s", unclassified, got)
	}
	_, unclassified = renderSynthetic(t, "ShutdownWorker", []Identity{Worker, Relay}, fields...)
	if len(unclassified) != 1 || !strings.Contains(unclassified[0], "[nava-relay]") {
		t.Errorf("rule for one identity only: unclassified = %v, want the relay's", unclassified)
	}
}

func TestRequestFieldWalkStopsAtCycles(t *testing.T) {
	got, _ := renderSynthetic(t, "Synthetic", []Identity{Relay}, baseFields()...)
	for _, row := range []string{
		"| `failure.cause → (Failure)` | `synthetic.v1.Failure` | cycle: its fields are listed above |",
		"| `failure.info.back → (Failure)` | `synthetic.v1.Failure` | cycle: its fields are listed above |",
		"| `failure.info.detail` | `string` | " + notIDOrQueue + " |",
		"| `tags[]` | `string` | " + notIDOrQueue + " |",
		"| `namespace` | `string` | " + classGate + " |",
	} {
		if !strings.Contains(got, row) {
			t.Errorf("missing row %s in:\n%s", row, got)
		}
	}

	// The real temporal.api.failure.v1.Failure, under every API that carries it.
	real, _ := renderRequestFieldGolden(d9AllowedRequests(t))
	for _, method := range []string{"RespondActivityTaskFailed", "RespondWorkflowTaskFailed"} {
		section := real[strings.Index(real, "\n## "+method+"\n"):]
		section = section[:strings.Index(section[1:], "\n## ")+1]
		if !strings.Contains(section, "| `failure.cause → (Failure)` | `temporal.api.failure.v1.Failure` |") {
			t.Errorf("%s: no failure.cause cycle marker", method)
		}
	}
}

// A google.protobuf.Any hides its body's fields from the walk, so an Any with
// no anyBodies entry fails even with a matching golden; the protocol message
// body is expanded into the three update bodies the server accepts.
func TestRequestFieldGoldenFailsOnUnexpandedAny(t *testing.T) {
	file := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("synthetic/v1/any.proto"),
		Package:    proto.String("synthetic.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/protobuf/any.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Request"), Field: []*descriptorpb.FieldDescriptorProto{
			scalarField("namespace", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			{Name: proto.String("body"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".google.protobuf.Any")},
		}}},
	}
	fd, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatalf("synthetic descriptor: %v", err)
	}
	got, unclassified := renderRequestFieldGolden([]allowedRequest{{method: "Synthetic", ids: []Identity{Worker}, input: fd.Messages().ByName("Request")}})
	problems := requestFieldProblems(got, got, unclassified)
	if len(problems) != 1 || !strings.Contains(problems[0], "Synthetic body is a google.protobuf.Any") {
		t.Errorf("problems = %v, want one unexpanded-Any failure", problems)
	}
	if !strings.Contains(got, "| `body` | `google.protobuf.Any` | **unexpanded `Any`** |") {
		t.Errorf("golden row does not flag the Any:\n%s", got)
	}

	real, _ := renderRequestFieldGolden(d9AllowedRequests(t))
	for _, row := range []string{
		"| `messages[].body (Acceptance)` | `temporal.api.update.v1.Acceptance` |",
		"| `messages[].body (Rejection).failure.child_workflow_execution_failure_info.workflow_execution.workflow_id` | `string` | " + classD9hFailure + " |",
		"| `messages[].body (Response).outcome.failure.child_workflow_execution_failure_info.workflow_execution.workflow_id` | `string` | " + classD9hFailure + " |",
		"| `messages[].body (Response).outcome.failure.cause → (Failure)` |",
	} {
		if strings.Count(real, row) != 2 { // RespondWorkflowTaskCompleted and RespondWorkflowTaskFailed
			t.Errorf("want row %s in both workflow task responses", row)
		}
	}
}
