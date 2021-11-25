@startuml

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
    + ScanPattern()
    + WriteCoverage()
    + ScanCoverage()
}

class CoverageScanner {
    - PatternSet store
    + Scan() Coverage
}

class Scanner {
    + Scan() Coverage[]
    - CoverageScanner[] scannerSet
}


class Merger {
    + Merge(Coverage[])
    - PatternSet store
}

object PipelineController

LevelDB --|> KV

PatternSet ..> LevelDB
CoverageScanner ..> PatternSet
Scanner ..> CoverageScanner
Merger  ..> PatternSet

Scanner .r. Merger

PipelineController --> Scanner: scan coverage
PipelineController --> Merger: merge and record coverage


together {
    class Scanner
    class Merger
}


@enduml