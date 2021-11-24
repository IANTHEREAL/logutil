@startuml

interface ObjectIterator {
    + iterator()
    + next()
    + hasNext()
}

interface PatternSet {
    + encodePattern()
    + writePattern()
}

class Repo {
    + ForEach()
    - GetRepoPath()
    - String repoRootPath
    - Pkg *pkgSet[]
}

class PackageCompilation  {
    - FileCompilation *fileSet[]
    - String pkgName
    + ForEach()
    + Compile()
}

class FileCompilation {
    + Compile()
    + RunAnalysis(Func)
    - String fileName[corpus/pkg/filename]
    - *ast
}

class Build {
    + Build()
}

class Traverse {
    + Visit()
}

class Filter {
    + Filter()
}

class FilePatternSet {
    + WritePattern()
}

object Controller

Repo --|> ObjectIterator
FilePatternSet --|> PatternSet

Repo o-right- PackageCompilation
PackageCompilation o-right- FileCompilation

Build ..> Repo
Traverse ..> Repo
Traverse ..> PackageCompilation
Filter ..> FileCompilation

Build .r. Traverse
Traverse .r. Filter
Filter .r. FilePatternSet

Controller --> Build: build AST
Controller --> Traverse: traverse
Controller --> Filter: filter log ast node
Controller --> FilePatternSet: encode and save


together {
    class Traverse
    class Filter
    class FilePatternSet
}


@enduml