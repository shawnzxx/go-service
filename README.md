## reference
https://github.com/ardanlabs/service4.1-video/commits/main?after=441f495cc8882d24d68767ea27cc4dbc1ebc5359+69&branch=main&qualified_name=refs%2Fheads%2Fmain

## Design Philosophy
1. don't make things easier to do, make thing easier to understand (write code in mentality for next person will maintian it)
1. Don't add complexity until we need it (you start with mono, then envolve to microservice)
1. everything we do must be precise
1. write code always start from readability then refactor to find a simplicity

## Deploy first and then development
start from day 1, for local development you shall try to setup and test your work on local with similar enviroment setup like in staging and production, things like we want to run our applications on k8s for local deployment

### project principal
folder structure not see more than 5 layers, this mgic number will keep you maintian mental model (this is what you understand code, review, code, talk/discuss your code)
### 5 layers
1. app layer
all things we are build for binary like services and tools their main.go will exist inside app layer
1. business layer
any package that we are building like business rules, business process will put inside this layer
1. foundation layer
eventually will move to it's own repo (company kit repo) or vendor folder, which currently we temp use for this folder and could techniqually use in another project
1. vendor layer
this layer hold for other 3rd party dependencies or company internal library which your project will be used
1. zarf layer
layer of code that could hold everything need for docker, k8s, build, deployment, configuration
### layers relationship
5 layers import shall only go down can't go up, if I see business layer tying to import app layer, code review stopped
app can import everything because is the most top layer in 5, business layer can import their sub folder package or down from foundation layer

## package
we define package for provide not contain, every package we defined must provide certain APIs which accept either concrete type of data or interface type of data, we don't want to see package name like util, helps, models, types
think about it, package is we define apis that create a firewall around itself, the onlly way that expose is through it's public apis
and every apis is data transformation

package need to have same package name go file, logger folder need to have logger.go, and logger package
one good way to identify package is contain instead of previde kind package: ask youself does it make sense to have file name call package.go? if doesn;t make sense to have package.go then you could be creating contain kind of package try to avoid it

### type system
a type system allow data input and output throught apis, we have two kind of type, concrete type or interface type 
- it allow input come into the api (concrete data or interface type)
- it allow output to come out

### api
the api can choose received data in two ways, it can use concrete type, you can also write apis accept interface type: the data not based what it is, but based on what it can do (polymorphism)
- what is polymorphism?
`A piece of code change it behivour depends on the type of concrete data it operating on`
you can write function to be polymorphism by saying I don't want concrete data by what it is, but I want concrete data based on what it can do


Deploy first mentality


structure logging convert human readable log
logger package should in foundation layer highly reusable across different project
you can not have common,util package, every package should have reason, provide somethings like net, http is good package
package provide or package common
filename try to discribe the purpose then this is smell, package should describe purpose

compile time polymorphism (generic) VS runtime polymorphism (interface)
always use runtime polymorphism
when you use reflect then consider change to compile time polymorphism

## configuration
### rules
- only place to read configuration is on main.go, and pass configuration down to other parts of program
- your application binary should allow to type "help" can print out all configuration options availiabe in your program, including default configuration, no singlton
- any default you have should be aloow to overwritten through env vairable or command line flags, you need to support both and command line flag shall rule them all
- default, default, default whoever clone the program down, need to run your program without any code change. except those Azure keys need to configured, but you need to write clear instruction how to do it in README.md file

## shutdown
concurrency VS parallel
concurrency means undefined out of order execution
parallel means you shall at least two threads and two cors running at same time

Goroutine general principal: parent G can not terminate when still have child G running, make sure all child G terminated before parent G terminated. 
one sample is: some people inside the handlder start a Go rountinue, then return the handler, then parent G terminated we still have child G running like orphan G

## Value VS Pointer
general rule: 
- if is pure data need to transfer in to the function, keep use value sementic
- if is API means could shared by all program to call, then pointer sememtic, like NewApp, NexContextMux() ect. those are all APIs
