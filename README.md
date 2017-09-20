# Dependency Resolver
A simple language independent dependency resolver that supports multiple registries to pull from.

In most cases you have the need to pull some dependencies (lib files for C++, JAR files for Java, ...) in order to build the software.
In order to keep my build scripts simple, I want to have a single file that defines all my dependencies and then run a simple command

    dep

in order to retrieve all dependencies. So I can fairly simple include this command in my build scripts.

## Configuration
In order to configure `dep`, create a file called `config.yml` or `config.toml` or `config.json` in one of the following directories:
 - $Home/.deprc
 - ./.deprc

Here is an example config file:

```yaml
# Where to find the 7z executable
SevenZipExecutable: /your/path/to/7zip
# The directory name where you want to extract your dependencies
DependencyDirectory: dep
# The repositories where to search for your dependencies
Repositories: [
  'https://some.de/url/endpoint',
  '/or/a/local/directory'
]
```

You then define your dependencies in a `.dep` file in your project directory:

```
# Each line defines a single dependency that should be resolved.
# A dependency is defined by a name and an version. If no version is
# given, the latest version will be installed.
mmf 2.1
pe 1.0
libdmtx 0.7.4
```

Then execute `dep` and all your dependencies will be downloaded in the `dep` folder.
