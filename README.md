
# Installation

    go get github.com/nvlled/gost

# Commandline usage

## Usage and Help
Running gost without any arguments shows
the list of actions and options available.

## Creating a new project
To create a new project, run
the following command:

    $ gost new sampel

This creates a directory named "sampel" in the current directory.
The created directory has enough files for building.


## Building the project
Actions (such as build) requires srcDir and destDir to be specified.
Options may be provided in the commandline:

    $ gost --srcDir sampel/src --destDir sampel/build build

Options may also be provided in the file:

    $ gost -opts sampel/gostopts build

Or both:

    # output dir is relative to sampel (i.e. sampel/output)
    $ gost -opts sampel/gostopts -destDir output build

Options in commandline takes priority.

You can run the previous command from any working directory.
The srcDir and destDir specified in the options file
are relative to the directory of the options file.

The format of the options file is the same as the one in the commandline:

    $ cat sampel/gostopts
    # (output ommited)

By default, gost searches for a file named gostopts for options
in the current directory.
So you can change directory in the project directory and run the command
without options:

    $ cd sampel
    $ gost build

# Project elements

## Envs
Envs are key-value pairs, where entries are separated by newlines
and key-values are seperated by colons:

    (Lines without colons are ignored, like this one)
    foo: bar
    baz: blah
    xyz: 1234

Envs have two purpose:
- to override default values that changes behaviour for a given action
- to provide values in the rendering context (of text/template)
  that is accesible via dot notation: {{.keyName}}

Envs are either embedded in html, js, or css files,
or they are put in a file named `env`.

    # in the sampel directory
    $ cat src/env
    $ cat src/index.html

When embedding envs, separators are used
to differentiate env entries from actual content.

Separators are consecutive dashes "------" that
are at least three characters long and
the beginning and ending separators must match
in length.

### Default Env values

The env file located in the src directory is called
base-env and can contain entries than can be
used to override default values:

    includes-dir: includes
    layouts-dir: layouts
    protos-dir: protos
    verbatim-files: somefile.html subsite/
    exclude-files: dir/ testfile

Files in the includes-dir contains snippets of
text that can be included in other files.
Each snippet must be explicitly defined:

	{{define "greet"}}
        <p>Hello .</p>
	{{end}}

	{{define "emphasize"}}
        <em><blink>__{{.}}__</blink><em>
	{{end}}

(See docs for text/template)

Files in the layouts-dir contains layouts
for other html files. Unlike in the includes-dir,
there is no need to explicitly define a name for each template.
The filename (including the file extension) will
be used as the name for the template.

Each layout must have a {{.contents}} in it.
For example:

    <html lang="en">
    <body>
    <div id="wrapper">
        {{.contents}}
    </div>
    <div id="footer">do not steal my unoriginal, derived ideas © copyfright 20XX</div>
    </body>
    </html>

Files in the protos-dir are used as prototypes for
creating new project files using the `newfile` action.
It uses a different delimeters ([[ and  ]]) for text/template actions.
No need to explicitly define a name as in the layouts-dir.


## Itemplates and rendering
Itemplates are files that are subject to rendering.
For the time being, itemplates are html, js or css files.
Only html files can have layout.

For each itemplate, there is an associated env that is accessible
using the dot notation, {{.keyName}}.

The entries of env for a given file depends on its
embedded env, the env file in the directory of given file
and all the env of the subdirectories up to the srcDir.

To make things concrete, consider the following:

    - src
      env
      - subdir/
        env
        - page.html

Suppose src/env contains

    x: 100
    y: aaa

and src/subdir/env contains

    x: 200
    z: nope
    title: some title

and src/subdir/page.html contains:

    ----------------
    x: 900
    title: Values of x, y, and z
    ----------------

    <h1>{{.title}}</h1>
    x = {{.x}}
    y = {{.y}}
    z = {{.z}}


The output of page.html when rendered would be:

    <h1>Values of x, y and z</h1>
    x = 900
    y = aaa
    z = nope


## Functions
In addition to the predifined global functions in the
text/template, several functions are available for use:

- url
- urlfor
- with_env
- genid
- shell

### url(path string) string
The returned value of url function depends on the value of
relative-url entry in the env. If relative-url is set to true,
it will convert absolute urls (urls starting with a slash, /like/this/one)
into relative urls. Relative urls remain relative.

Otherwise, if relative-url is set to false or 0, the path argument
is returned as is. Note that relative urls will not be converted
to absolute urls.

### urlfor(id string) string
Returns the url for the file that has an id entry in the env
that matches with given id. The return value is also determined
by the value of relative-url env entry. See url function.

### with_env(envKey, envValue string) []env
Returns the a list of envs of files that contains the given
env entries. This list of envs can be iterated using the range
function in the text/template package:

    {{range (with_env "category" "blog")}
        <p>{{.title}}</p>
    {{end}}

The example code above will output all the files that has
an env entries "category: blog".

### genid() string
Returns a random string. Used for prototypes of files.

### shell(command) string
Executes a shell command and returns the output of the command.


# Notes
- The reader/user is familiar with using the commandline interface.
  At the very least, you should know what the dollar sign means
  (and it doesn't involve monies)

- Familiarity with the text/template package included in the standard library
  is also required

- Tested yet only on linux (may not work on windows since some path separators are hardcoded)
