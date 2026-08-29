package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: ghosttag <image.jpg|image.jpeg|image.png>")
		os.Exit(2)
	}
	r, err := inspect(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ghosttag:", clean(err.Error()))
		os.Exit(1)
	}
	fmt.Printf("ghosttag — image metadata report\n\nFile\n  Name: %s\n  Detected format: %s\n  Size: %d bytes\n  Dimensions: %d × %d pixels\n  SHA-256: %s\n\nMetadata\n", r.name, r.format, r.size, r.width, r.height, r.hash)
	if len(r.m["containers"]) > 0 {
		fmt.Printf("  Containers: %s\n", strings.Join(r.m["containers"], ", "))
	}
	for _, s := range []struct{ label, key string }{{"Location", "location"}, {"Capture time", "capture"}, {"Device make", "make"}, {"Device model", "model"}, {"Author or copyright", "author"}, {"Comment or description", "comments"}} {
		show(s.label, r.m[s.key])
	}
	cats := r.m.categories()
	fmt.Println("\nPrivacy context")
	if len(cats) == 0 {
		fmt.Println("  No supported privacy-relevant metadata categories were found.\n  This does not prove the image is anonymous.")
		return
	}
	fmt.Printf("  Categories found (%d): %s\n", len(cats), strings.Join(cats, ", "))
	if len(cats) >= 3 {
		fmt.Printf("  Note: This file contains %d privacy-relevant metadata categories: %s. In combination, these details can reveal more context than each detail alone. Consider whether they are appropriate for the intended recipient or platform.\n", len(cats), strings.Join(cats, ", "))
	}
}

func show(label string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Printf("  %s:\n", label)
	for _, v := range values {
		fmt.Printf("    - %s\n", v)
	}
}
