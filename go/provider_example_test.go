package container

func ExampleWithName() {
	// WithName usage is covered by the corresponding triplet tests.
}

func ExampleWithMemory() {
	// WithMemory usage is covered by the corresponding triplet tests.
}

func ExampleWithCPUs() {
	// WithCPUs usage is covered by the corresponding triplet tests.
}

func ExampleWithDetach() {
	// WithDetach usage is covered by the corresponding triplet tests.
}

func ExampleWithPorts() {
	// WithPorts usage is covered by the corresponding triplet tests.
}

func ExampleWithVolumes() {
	// WithVolumes usage is covered by the corresponding triplet tests.
}

func ExampleWithArgs() {
	// WithArgs sets the container command/args, appended after the image.
	_ = ApplyRunOptions(WithArgs("sleep", "300"))
}

func ExampleApplyRunOptions() {
	// ApplyRunOptions usage is covered by the corresponding triplet tests.
}
