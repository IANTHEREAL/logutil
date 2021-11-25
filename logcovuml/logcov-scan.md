@startuml

interface KV {
    + Write(key, value)
    + Get(key) value
    + Scan(range) 
}

interface LogReader {
    + Scan() []byte
}

interface LogParser {
    + Parse() Log
    + IsThisLog()
}

class FileReader {
    + Scan() []byte
    - FD reader
    - String logPath
}

class ZapLogParser {
    + Parse() Log
    + IsLog() bool
}

class LogScanner {
    + scan()
    - String logPath
    - LogParser parser
    - LogReader reader
}

class Pipeline {
    + Run()
    - LogScanner[]  scannerSet
    - PatternMatcher parser
    - LogCoverager coverager 
}

object Position {
    + String file
    + Int32  lineNo
}

object Log {
    + String path
    + Position pos
    + String logLevel
    + String content
}

class PatternMatcher {
    + Match(Log) Pattern[]
    - PatternTrie trie
}

Class PatternTrie {
    + Insert(LogSignature, Pattern)
    + Match(Log)
    - TreeNode *root
}

class LogCoverager {
    + Compute(Log, Pattern)
    - FilePatternSet patternSet
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
}


LevelDB --|> KV
FileReader --|> LogReader
ZapLogParser --|> LogParser

PatternSet o-- LevelDB
LogScanner o-- FileReader
LogScanner o-- ZapLogParser
Log o-- Position


Pipeline o-- LogScanner: scan runtime logs
Pipeline o-- PatternMatcher: Match log base on log pattern
Pipeline o-- LogCoverager: record log coverage

PatternMatcher o-- PatternTrie
LogCoverager o-- PatternSet


LogScanner .r. PatternMatcher
PatternMatcher .r. LogCoverager


together {
    class LogScanner
    class PatternMatcher
    class LogCoverager
}


@enduml