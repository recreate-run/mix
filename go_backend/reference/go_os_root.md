# Traversal-resistant file APIs

A *path traversal vulnerability* arises when an attacker can trick a program
into opening a file other than the one it intended.  
This post explains this class of vulnerability, some existing defenses against it,  
and describes how the new [`os.Root`](/pkg/os#Root) API added in Go 1.24 provides
a simple and robust defense against unintentional path traversal.

## Path traversal attacks

“Path traversal” covers a number of related attacks following a common pattern:  
A program attempts to open a file in some known location, but an attacker causes
it to open a file in a different location.

If the attacker controls part of the filename, they may be able to use relative
directory components (`..`) to escape the intended location:

```go
f, err := os.Open(filepath.Join(trustedLocation, "../../../../etc/passwd"))
````

On Windows systems, some names have special meaning:

```go
// f will print to the console.
f, err := os.Create(filepath.Join(trustedLocation, "CONOUT$"))
```

If the attacker controls part of the local filesystem, they may be able to use
symbolic links to cause a program to access the wrong file:

```go
// Attacker links /home/user/.config to /home/otheruser/.config:
err := os.WriteFile("/home/user/.config/foo", config, 0o666)
```

If the program defends against symlink traversal by first verifying that the intended file
does not contain any symlinks, it may still be vulnerable to
[time-of-check/time-of-use (TOCTOU) races](https://en.wikipedia.org/wiki/Time-of-check_to_time-of-use),
where the attacker creates a symlink after the program’s check:

```go
// Validate the path before use.
cleaned, err := filepath.EvalSymlinks(unsafePath)
if err != nil {
  return err
}
if !filepath.IsLocal(cleaned) {
  return errors.New("unsafe path")
}

// Attacker replaces part of the path with a symlink.
// The Open call follows the symlink:
f, err := os.Open(cleaned)
```

Another variety of TOCTOU race involves moving a directory that forms part of a path
mid-traversal. For example, the attacker provides a path such as `a/b/c/../../etc/passwd`,
and renames `a/b/c` to `a/b` while the open operation is in progress.

---

## Path sanitization

Before we tackle path traversal attacks in general, let’s start with path sanitization.
When a program’s threat model does not include attackers with access to the local file system,
it can be sufficient to validate untrusted input paths before use.

Unfortunately, sanitizing paths can be surprisingly tricky,
especially for portable programs that must handle both Unix and Windows paths.
For example, on Windows `filepath.IsAbs(`\foo`)` reports `false`,
because the path `\foo` is relative to the current drive.

In Go 1.20, we added the [`path/filepath.IsLocal`](/pkg/path/filepath#IsLocal) function,
which reports whether a path is “local”. A “local” path is one which:

* does not escape the directory in which it is evaluated (`../etc/passwd` is not allowed);
* is not an absolute path (`/etc/passwd` is not allowed);
* is not empty (`""` is not allowed);
* on Windows, is not a reserved name (`COM1` is not allowed).

In Go 1.23, we added the [`path/filepath.Localize`](/pkg/path/filepath#Localize) function,
which converts a `/`-separated path into a local operating system path.

Programs that accept and operate on potentially attacker-controlled paths should almost
always use `filepath.IsLocal` or `filepath.Localize` to validate or sanitize those paths.

---

## Beyond sanitization

Path sanitization is not sufficient when attackers may have access to part of
the local filesystem.

Multi-user systems are uncommon these days, but attacker access to the filesystem
can still occur in a variety of ways.
An unarchiving utility that extracts a tar or zip file may be induced
to extract a symbolic link and then extract a file name that traverses that link.
A container runtime may give untrusted code access to a portion of the local filesystem.

Programs may defend against unintended symlink traversal by using the
[`path/filepath.EvalSymlinks`](/pkg/path/filepath#EvalSymlinks) function to resolve links
in untrusted names before validation, but as described above this two-step process
is vulnerable to TOCTOU races.

Before Go 1.24, the safer option was to use a package such as
[github.com/google/safeopen](/pkg/github.com/google/safeopen),
that provides path traversal-resistant functions for opening a potentially-untrusted
filename within a specific directory.

---

## Introducing `os.Root`

In Go 1.24, we are introducing new APIs in the `os` package to safely open
a file in a location in a traversal-resistant fashion.

The new [`os.Root`](/pkg/os#Root) type represents a directory somewhere
in the local filesystem. Open a root with the [`os.OpenRoot`](/pkg/os#OpenRoot) function:

```go
root, err := os.OpenRoot("/some/root/directory")
if err != nil {
  return err
}
defer root.Close()
```

`Root` provides methods to operate on files within the root.
These methods all accept filenames relative to the root,
and disallow any operations that would escape from the root either
using relative path components (`..`) or symlinks.

```go
f, err := root.Open("path/to/file")
```

`Root` permits relative path components and symlinks that do not escape the root.
For example, `root.Open("a/../b")` is permitted. Filenames are resolved using the
semantics of the local platform.

`Root` currently provides the following set of operations:

```go
func (*Root) Create(string) (*File, error)
func (*Root) Lstat(string) (fs.FileInfo, error)
func (*Root) Mkdir(string, fs.FileMode) error
func (*Root) Open(string) (*File, error)
func (*Root) OpenFile(string, int, fs.FileMode) (*File, error)
func (*Root) OpenRoot(string) (*Root, error)
func (*Root) Remove(string) error
func (*Root) Stat(string) (fs.FileInfo, error)
```

In addition to the `Root` type, the new [`os.OpenInRoot`](/pkg/os#OpenInRoot) function
provides a simple way to open a potentially-untrusted filename within a
specific directory:

```go
f, err := os.OpenInRoot("/some/root/directory", untrustedFilename)
```

The `Root` type provides a simple, safe, portable API for operating with untrusted filenames.

---

## Caveats and considerations

### Unix

* Implemented using `openat` system calls.
* Tracks root directory across renames/deletion.
* Defends against symlinks, not against mount points (e.g., bind mounts).

### Windows

* Uses directory handle, preventing rename/delete until closed.
* Blocks reserved names like `NUL`, `COM1`.

### WASI

* Uses WASI preview 1 FS API.
* Traversal safety depends on WASI implementation.

### GOOS=js

* Uses Node.js FS API.
* Vulnerable to TOCTOU races in symlink validation.
* Tracks by directory name, not FD.

### Plan 9

* No symlinks.
* Performs lexical sanitization of filenames.

### Performance

* `Root` operations can be slower than non-`Root`.
* Use `filepath.Clean` to reduce `..` components.

---

## Who should use os.Root?

You should use `os.Root` or `os.OpenInRoot` if:

* You are opening a file in a directory; **AND**
* The operation should not access a file outside that directory.

**Example**: An archive extractor writing files to an output directory.

But a CLI tool writing to a user-specified location **should not** use `os.Root`.

**Bad:**

```go
// Might open a file not located in baseDirectory.
f, err := os.Open(filepath.Join(baseDirectory, filename))
```

**Good:**

```go
// Only opens files under baseDirectory.
f, err := os.OpenInRoot(baseDirectory, filename)
```

---

## Future work

The `os.Root` API is new in Go 1.24.
We expect additions and refinements in future releases.

* Performance improvements (e.g., Linux `openat2`).
* Support for more FS operations (e.g., symlinks, renames).
  See [go.dev/issue/67002](/issue/67002).

---
