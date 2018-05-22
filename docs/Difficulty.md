Solving this problem effectively in an acceptable manner is difficult, as most of the straight-forward solutions have
significant drawbacks. The following outlines some of the possible approaches to this problem and their respective
drawbacks.

## Embed other binaries directly
One approach to solve the issue is by embedding the compiled binaries of the dependent programs directly in the source
code as a binary or a resource. In Go, this can be done using a tool such as
[go-bindata](https://github.com/jteeuwen/go-bindata).The process would look something like this:

* Compile all of the desired binaries
* Encode the compiled binaries and compile them into the compiled program's source code
* When the combined program is run and a sub-program's functionality is needed, write the sub-program to disk an execute
  it

This approach has several drawbacks:

* All of the sub-programs must be compiled separately
* Because the binaries of the sub-programs are embedded directly in the over-all program, the size of the over-all
  program will be at least the sum of the size of all of the compiled programs, which can be extremely large
* Running the sub-programs requires the ability to write the sub-program to some location on disk and execute it
  * This may be difficult/not possible depending on the execution environment

## Embed the source code for the other programs
Another approach is to embed all of the source code for the dependent programs as assets in the over-all program. The
process looks something like this:

* Embed the source code of all of the dependent programs in the program
* When the combined program is run and sub-program functionality is needed:
  * Write the embedded source code to disk
  * Compile the source code into an executable
  * Run the program that was compiled

This approach results in a binary that is much smaller than the previous approach because source code takes up less
space than binaries and the common library code is omitted. However, it still has many drawbacks:

* The system on which the program is run must have build tools to compile source code (this is often not the case)
* Compiling source code can take a long time, which impacts the performance of the over-all program
* Has the same issue as the binaries approach with regards to being able to write and execute other binaries

## Manually write a program that calls into the other programs
If the source code for the other programs is available, one option is to integrate those programs into the main program
by hand. The process would look something like this:

* Identify all of the source code required for the dependent programs and add them to the over-all program
* Convert all of the "main" functions into library functions
* Resolve conflicts between dependent libraries
* Determine any shared global state that is accessed and resolve them
* Create a way for the over-all program to call into the library code

This approach has the advantage of creating a single compiled executable that is relatively small and can be run on its
own. However, there are still significant drawbacks:

* Error-prone because it requires manually copying, modifying and deconflicting code
* It can be extremely hard to update the code later if new versions of the dependent programs are released
* Many programs are not designed to be run in-process -- they may use shared global state or make calls like `os.Exit`
  that terminate the entire program. Because of this, even when converted to be a library, the programs may not be safe
  to run.
