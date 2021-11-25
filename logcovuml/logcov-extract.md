@startuml

interface Traverser {
    + Visit()
}

interface KV {
    + Write(key, value)
    + Get(key) value
    + Scan(range) 
}

Class LevelDB {
    + Write(key, value)
    + Get(key) value
    + Scan(range) 
}

Class PatternSet {
    - KV kv
    + WritePattern()
}

class Repo {
    + Visit()
    - GetRepoPath()
    - String repoRoot
    - PackageCompilation *pkgSet[]
}

class PackageCompilation  {
    - FileCompilation *fileSet[]
    - String pkgName
    + Visit()
    + Compile()
}

class FileCompilation {
    + Compile()
    + RunAnalysis(func)
    - String fileName[repo/pkg/filename]
    - *ast
}

class Builder {
    + Build(repoPath) *Repo
}

class Vistor {
    + Visit()
}

class Filter {
    + Filter()
}

object Controller

Repo --|> Traverser
PackageCompilation --|> Traverser
LevelDB --|> KV

PatternSet o-right- LevelDB
Repo o-right- PackageCompilation
PackageCompilation o-right- FileCompilation

Builder ..> Repo
Vistor ..> Repo
Vistor ..> PackageCompilation
Filter ..> FileCompilation

Builder .r. Vistor
Vistor .r. Filter
Filter .r. PatternSet

Controller --> Builder: build AST
Controller --> Vistor: traverse
Controller --> Filter: filter log ast node
Controller --> PatternSet: encode and save


together {
    class Vistor
    class Filter
    class PatternSet
}


@enduml