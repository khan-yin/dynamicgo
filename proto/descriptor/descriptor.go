package descriptor

import (
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

type Descriptor struct {
	*descriptorpb.DescriptorProto
}

type FileDescriptor struct {
	*descriptorpb.FileDescriptorProto
}

type ServiceDescriptor struct {
	*descriptorpb.ServiceDescriptorProto
}

type MethodDescriptor struct {
	*descriptorpb.MethodDescriptorProto
}

type MessageDescriptor struct {
	*descriptorpb.DescriptorProto
}

type FieldDescriptor struct {
	*descriptorpb.FieldDescriptorProto
}


// special fielddescriptor for enum, oneof, extension, extensionrange

type EnumDescriptor struct {
	*descriptorpb.EnumDescriptorProto
}

type EnumValueDescriptor struct {
	*descriptorpb.EnumValueDescriptorProto
}

type OneofDescriptor struct {
	*descriptorpb.OneofDescriptorProto
}

// ExtensionDescriptor is the same as FieldDescriptor
type ExtensionDescriptor = FieldDescriptor

type ExtensionRangeDescriptor struct {
	*descriptorpb.DescriptorProto_ExtensionRange
}

