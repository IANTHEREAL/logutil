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

class Corpus {
    + iterator()
    + next() File
    + hasNext()
    - constructAST()
    - String corpusName
    - Pkg *pkgSet[]
}
note bottom of Corpus: Corpus is a project repo contains multiple packages

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
    + build()
}

class Traverse {
    + walk()
}

class Filter {
    + filter(File)
}

class FilePatternSet {
    + writePattern()
}

object Controller

Corpus --|> ObjectIterator
FilePatternSet --|> PatternSet

Corpus o-right- PackageCompilation
PackageCompilation o-right- FileCompilation

Build ..> Corpus
Traverse ..> Corpus
Traverse ..> PackageCompilation

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