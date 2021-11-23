@startuml

interface LogScanner {
    + iterator()
    + next()
    + hasNext()
}

interface PatternSet {
    + encodePattern()
    + writePattern()
    + matchPattern()
    + encodeCoverage()
    + recordCoverage()
    + CreateLogScanner()
}

class FilePatternSet {
    + encodePattern()
    + writePattern()
    + matchPattern()
    + encodeCoverage()
    + recordCoverage()
    + CreateLogScanner()
    - String filePath
}

class FileLogScanner {
    + iterator()
    + next()
    + hasNext()
}

class Scanner {
    + scan()
    - FilePatternSet patternSet
}


class Merger {
    + merge()
    + record()
    - FilePatternSet patternSet
}


object PipelineController

PatternSet ..> LogScanner
FilePatternSet ..> FileLogScanner

FilePatternSet --|> PatternSet
FileLogScanner --|> LogScanner


Merger o-d- FilePatternSet
Scanner o-d- FilePatternSet

Scanner .r. Merger

PipelineController --> Scanner: scan coverage
PipelineController --> Merger: merge and record coverage


together {
    class Scanner
    class Merger
}


@enduml