# generate-interface-mocks

This tools generates Golang interfaces using [this prompt](./prompts/generate-go-mock-implementations.md). I've played around a lot with the prompt. I'm sure it could be improved and made more succinct, however it does the trick and I'm happy enough with it!

## Usage

It currently requires `OPENAI_API_KEY` to be set in the environment.

```sh
# llm-tools generate-interface-mocks -h
NAME:
   llm-tools generate-interface-mocks - Generates mock implementations for Go interfaces

USAGE:
   llm-tools generate-interface-mocks [command options] [arguments...]

OPTIONS:
   --package value         the package name to generate mocks for
   --interface-file value  filepath of file to generate mocks for
   --output-folder value   folder to output generated mocks to
   --model value           GhatGPT model to use for prompts. gpt-3.5-turbo seems more than enough. (default: "gpt-3.5-turbo")
   --openai-api-key value  OpenAI API key, will read from environment variables or from config file [$OPENAI_API_KEY]
   --help, -h              show help

```

Example:

```sh
# Set your openapi API key once
llm-tools -debug config set --openai-api-key sk-...

# Generate mock implementations
llm-tools generate-interface-mocks \
   -package=github.com/kristofferostlund/some-app/domain \
   -interface-file=/home/kristoffer/dev/some-app/domain/interfaces.go \
   -output-folder=/home/kristoffer/dev/some-app/domain/mock

```

## Prompting techniques

### Provide an example through single-shot prompting

See single-shot [input](./prompts/single-shot.example-input.md) and [output](./prompts/single-shot.example-output.md).

I found this **very** powerful: the single most important trick for this particular case. I saw great success in consistency when providing an example of what is expected given a known, somewhat complex input.

The first versions of the one-shot prompt would sometimes make mistakes with the `filepath:` part of the schema. When I added a second declaration to the input and output that cleared up quite well.

It also helped overcome some really funky type errors where it previously would misunderstand how slice types works and was a bit eager with the package name (e.g. `package.[]Type` 🙃).

I haven't bothered trying, but perhaps I can reduce the main prompt by introducing an example for the in-prompt example of function types?

### Tell it to _shut up_ and follow the schema

Since I'm parsing the output, or well, matching a small set of known lines, I need it to follow a set schema. The single-shot prompt sure helped **a lot**, however before introducing that I had to tell it to _shut up_ as it from time to time would be eager to explain the code to me, other times it would echo the input, sometimes freestyle with the format (name in comment, name at the bottom, skip the name etc).

### Use a delimiter for distinct parts

> Again with the schema...

To be able to parse the output it must be consistent. I saw some (conceived) stability improvement when putting the template within a _delimited_ of triple quotes (`"""`), and of course explaining I did so.

### Giving it a persona

_You are a coding assistant ..._

This doesn't seem to have much impact on the output in this case. I've left it in for now, though mostly because it is of some help for me (and I find it a bit amusing).

### Explain things it did wrong and provide examples on what's expected

I've currently tried this with three different files using _my exact style_, all pretty consistent and mostly clutter free (or like, basically nothing but interface declarations). Not a massive sample size. So uh, your mileage may vary! 🤷

One thing it struggled with initially was the type check (`var _ appname.UserProvider = (*UserProvider)(nil)`). It would sometimes put it in the wrong place, or not at all. I found that it helped to explain in detail what I wanted it to do with it.
