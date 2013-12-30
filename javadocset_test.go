package javadocset

import(
    "testing"
    "os"
)

func TestNonExistentJavadocPathFails(t *testing.T) {
    err := Build("/i/dont/exist", "foo")
    if err == nil || err.Error() != "Javadoc path does not exist" {
        t.Error("Didn't fail properly. Got error ", err)
    }
}

func TestExistingDocsetPathFails(t *testing.T) {
    err := Build(os.TempDir(), os.TempDir())
    if err == nil || err.Error() != "Docset output path should not exist" {
        t.Error("Didn't fail properly. Got error ", err)
    }
}

