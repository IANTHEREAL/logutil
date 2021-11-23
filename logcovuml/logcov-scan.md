@startuml

interface ObjectIterator {
    + iterator()
    + next()
    + hasNext()
}

interface Reader {
    + scanLine()
}

interface PatternSet {
    + encodePattern()
    + writePattern()
    + matchPattern()
    + encodeCoverage()
    + recordCoverage()
}

class FileReader {
    + scanLine()
}

class LogReader {
    + ScanOneLog() Log
    - FileReader reader
    - String uniqueLogPath
}

class LogSet {
    + iterator()
    + next() Log
    + hasNext()
    - LogReader *logs[]
}

object CodePosition {
    + String file
    + Int32  lineNo
}

object Log {
    + String path
    + CodePosition pos
    + String logLevel
    + String content
}

class Scanner {
    + scan()
    - LogSet logs

}

class Matcher {
    + match()
    - FilePatternSet patternSet
}

class Coverage {
    + record()
    - FilePatternSet patternSet
}

class FilePatternSet {
    + encodePattern()
    + writePattern()
    + matchPattern()
    + encodeCoverage()
    + recordCoverage()
    - String filePath
}

object PipelineController

LogSet --|> ObjectIterator
FileReader -r-|> Reader
LogSet o-r- LogReader
LogReader o-- FileReader

FilePatternSet -r-|> PatternSet
Matcher o-d- FilePatternSet
Coverage o-d- FilePatternSet
Log o-d- CodePosition

Scanner o-d- LogSet

Scanner .r. Matcher
Matcher .r. Coverage

PipelineController --> Scanner: scan runtime logs
PipelineController --> Matcher: Match log base on log pattern
PipelineController --> Coverage: record log coverage


together {
    class Scanner
    class Matcher
    class Coverage
}


@enduml