
package imports

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/ast/astutil"
)

var Debug = false

var (
	inTests = false      
	testMu  sync.RWMutex 
)

var LocalPrefix string

var importToGroup = []func(importPath string) (num int, ok bool){
	func(importPath string) (num int, ok bool) {
		if LocalPrefix != "" && strings.HasPrefix(importPath, LocalPrefix) {
			return 3, true
		}
		return
	},
	func(importPath string) (num int, ok bool) {
		if strings.HasPrefix(importPath, "appengine") {
			return 2, true
		}
		return
	},
	func(importPath string) (num int, ok bool) {
		if strings.Contains(importPath, ".") {
			return 1, true
		}
		return
	},
}

func importGroup(importPath string) int {
	for _, fn := range importToGroup {
		if n, ok := fn(importPath); ok {
			return n
		}
	}
	return 0
}

type packageInfo struct {
	Globals map[string]bool 
}

func dirPackageInfo(srcDir, filename string) (*packageInfo, error) {
	considerTests := strings.HasSuffix(filename, "_test.go")

	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return &packageInfo{}, nil
		}
		return nil, err
	}

	fileBase := filepath.Base(filename)
	packageFileInfos, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return nil, err
	}

	info := &packageInfo{Globals: make(map[string]bool)}
	for _, fi := range packageFileInfos {
		if fi.Name() == fileBase || !strings.HasSuffix(fi.Name(), ".go") {
			continue
		}
		if !considerTests && strings.HasSuffix(fi.Name(), "_test.go") {
			continue
		}

		fileSet := token.NewFileSet()
		root, err := parser.ParseFile(fileSet, filepath.Join(srcDir, fi.Name()), nil, 0)
		if err != nil {
			continue
		}

		for _, decl := range root.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				info.Globals[valueSpec.Names[0].Name] = true
			}
		}
	}
	return info, nil
}

func fixImports(fset *token.FileSet, f *ast.File, filename string) (added []string, err error) {

	refs := make(map[string]map[string]bool)

	decls := make(map[string]*ast.ImportSpec)

	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	srcDir := filepath.Dir(abs)
	if Debug {
		log.Printf("fixImports(filename=%q), abs=%q, srcDir=%q ...", filename, abs, srcDir)
	}

	var packageInfo *packageInfo
	var loadedPackageInfo bool

	var visitor visitFn
	visitor = visitFn(func(node ast.Node) ast.Visitor {
		if node == nil {
			return visitor
		}
		switch v := node.(type) {
		case *ast.ImportSpec:
			if v.Name != nil {
				decls[v.Name.Name] = v
				break
			}
			ipath := strings.Trim(v.Path.Value, `"`)
			if ipath == "C" {
				break
			}
			local := importPathToName(ipath, srcDir)
			decls[local] = v
		case *ast.SelectorExpr:
			xident, ok := v.X.(*ast.Ident)
			if !ok {
				break
			}
			if xident.Obj != nil {

				break
			}
			pkgName := xident.Name
			if refs[pkgName] == nil {
				refs[pkgName] = make(map[string]bool)
			}
			if !loadedPackageInfo {
				loadedPackageInfo = true
				packageInfo, _ = dirPackageInfo(srcDir, filename)
			}
			if decls[pkgName] == nil && (packageInfo == nil || !packageInfo.Globals[pkgName]) {
				refs[pkgName][v.Sel.Name] = true
			}
		}
		return visitor
	})
	ast.Walk(visitor, f)

	unusedImport := map[string]string{}
	for pkg, is := range decls {
		if refs[pkg] == nil && pkg != "_" && pkg != "." {
			name := ""
			if is.Name != nil {
				name = is.Name.Name
			}
			unusedImport[strings.Trim(is.Path.Value, `"`)] = name
		}
	}
	for ipath, name := range unusedImport {
		if ipath == "C" {

			continue
		}
		astutil.DeleteNamedImport(fset, f, name, ipath)
	}

	for pkgName, symbols := range refs {
		if len(symbols) == 0 {

			delete(refs, pkgName)
		}
	}

	searches := 0
	type result struct {
		ipath string 
		name  string 
		err   error
	}
	results := make(chan result)
	for pkgName, symbols := range refs {
		go func(pkgName string, symbols map[string]bool) {
			ipath, rename, err := findImport(pkgName, symbols, filename)
			r := result{ipath: ipath, err: err}
			if rename {
				r.name = pkgName
			}
			results <- r
		}(pkgName, symbols)
		searches++
	}
	for i := 0; i < searches; i++ {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}
		if result.ipath != "" {
			if result.name != "" {
				astutil.AddNamedImport(fset, f, result.name, result.ipath)
			} else {
				astutil.AddImport(fset, f, result.ipath)
			}
			added = append(added, result.ipath)
		}
	}

	return added, nil
}

var importPathToName func(importPath, srcDir string) (packageName string) = importPathToNameGoPath

func importPathToNameBasic(importPath, srcDir string) (packageName string) {
	return path.Base(importPath)
}

func importPathToNameGoPath(importPath, srcDir string) (packageName string) {

	if pkg, ok := stdImportPackage[importPath]; ok {
		return pkg
	}

	pkgName, err := importPathToNameGoPathParse(importPath, srcDir)
	if Debug {
		log.Printf("importPathToNameGoPathParse(%q, srcDir=%q) = %q, %v", importPath, srcDir, pkgName, err)
	}
	if err == nil {
		return pkgName
	}
	return importPathToNameBasic(importPath, srcDir)
}

func importPathToNameGoPathParse(importPath, srcDir string) (packageName string, err error) {
	buildPkg, err := build.Import(importPath, srcDir, build.FindOnly)
	if err != nil {
		return "", err
	}
	d, err := os.Open(buildPkg.Dir)
	if err != nil {
		return "", err
	}
	names, err := d.Readdirnames(-1)
	d.Close()
	if err != nil {
		return "", err
	}
	sort.Strings(names) 
	var lastErr error
	var nfile int
	for _, name := range names {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		nfile++
		fullFile := filepath.Join(buildPkg.Dir, name)

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, fullFile, nil, parser.PackageClauseOnly)
		if err != nil {
			lastErr = err
			continue
		}
		pkgName := f.Name.Name
		if pkgName == "documentation" {

			continue
		}
		if pkgName == "main" {

			continue
		}
		return pkgName, nil
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no importable package found in %d Go files", nfile)
}

var stdImportPackage = map[string]string{} 

func init() {

	for _, pkg := range stdlib {
		if _, ok := stdImportPackage[pkg]; !ok {
			stdImportPackage[pkg] = path.Base(pkg)
		}
	}
}

var (

	scanGoRootOnce sync.Once

	scanGoPathOnce sync.Once

	populateIgnoreOnce sync.Once
	ignoredDirs        []os.FileInfo

	dirScanMu sync.RWMutex
	dirScan   map[string]*pkg 
)

type pkg struct {
	dir             string 
	importPath      string 
	importPathShort string 
}

type byImportPathShortLength []*pkg

func (s byImportPathShortLength) Len() int { return len(s) }
func (s byImportPathShortLength) Less(i, j int) bool {
	vi, vj := s[i].importPathShort, s[j].importPathShort
	return len(vi) < len(vj) || (len(vi) == len(vj) && vi < vj)

}
func (s byImportPathShortLength) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type gate chan struct{}

func (g gate) enter() { g <- struct{}{} }
func (g gate) leave() { <-g }

var visitedSymlinks struct {
	sync.Mutex
	m map[string]struct{}
}

func populateIgnore() {
	for _, srcDir := range build.Default.SrcDirs() {
		if srcDir == filepath.Join(build.Default.GOROOT, "src") {
			continue
		}
		populateIgnoredDirs(srcDir)
	}
}

func populateIgnoredDirs(path string) {
	file := filepath.Join(path, ".goimportsignore")
	slurp, err := ioutil.ReadFile(file)
	if Debug {
		if err != nil {
			log.Print(err)
		} else {
			log.Printf("Read %s", file)
		}
	}
	if err != nil {
		return
	}
	bs := bufio.NewScanner(bytes.NewReader(slurp))
	for bs.Scan() {
		line := strings.TrimSpace(bs.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		full := filepath.Join(path, line)
		if fi, err := os.Stat(full); err == nil {
			ignoredDirs = append(ignoredDirs, fi)
			if Debug {
				log.Printf("Directory added to ignore list: %s", full)
			}
		} else if Debug {
			log.Printf("Error statting entry in .goimportsignore: %v", err)
		}
	}
}

func skipDir(fi os.FileInfo) bool {
	for _, ignoredDir := range ignoredDirs {
		if os.SameFile(fi, ignoredDir) {
			return true
		}
	}
	return false
}

func shouldTraverse(dir string, fi os.FileInfo) bool {
	path := filepath.Join(dir, fi.Name())
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		return false
	}
	ts, err := os.Stat(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	if !ts.IsDir() {
		return false
	}
	if skipDir(ts) {
		return false
	}

	realParent, err := filepath.EvalSymlinks(dir)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return false
	}
	realPath := filepath.Join(realParent, fi.Name())
	visitedSymlinks.Lock()
	defer visitedSymlinks.Unlock()
	if visitedSymlinks.m == nil {
		visitedSymlinks.m = make(map[string]struct{})
	}
	if _, ok := visitedSymlinks.m[realPath]; ok {
		return false
	}
	visitedSymlinks.m[realPath] = struct{}{}
	return true
}

var testHookScanDir = func(dir string) {}

var scanGoRootDone = make(chan struct{}) 

func scanGoRoot() {
	go func() {
		scanGoDirs(true)
		close(scanGoRootDone)
	}()
}

func scanGoPath() { scanGoDirs(false) }

func scanGoDirs(goRoot bool) {
	if Debug {
		which := "$GOROOT"
		if !goRoot {
			which = "$GOPATH"
		}
		log.Printf("scanning " + which)
		defer log.Printf("scanned " + which)
	}
	dirScanMu.Lock()
	if dirScan == nil {
		dirScan = make(map[string]*pkg)
	}
	dirScanMu.Unlock()

	for _, srcDir := range build.Default.SrcDirs() {
		isGoroot := srcDir == filepath.Join(build.Default.GOROOT, "src")
		if isGoroot != goRoot {
			continue
		}
		testHookScanDir(srcDir)
		walkFn := func(path string, typ os.FileMode) error {
			dir := filepath.Dir(path)
			if typ.IsRegular() {
				if dir == srcDir {

					return nil
				}
				if !strings.HasSuffix(path, ".go") {
					return nil
				}
				dirScanMu.Lock()
				if _, dup := dirScan[dir]; !dup {
					importpath := filepath.ToSlash(dir[len(srcDir)+len("/"):])
					dirScan[dir] = &pkg{
						importPath:      importpath,
						importPathShort: vendorlessImportPath(importpath),
						dir:             dir,
					}
				}
				dirScanMu.Unlock()
				return nil
			}
			if typ == os.ModeDir {
				base := filepath.Base(path)
				if base == "" || base[0] == '.' || base[0] == '_' ||
					base == "testdata" || base == "node_modules" {
					return filepath.SkipDir
				}
				fi, err := os.Lstat(path)
				if err == nil && skipDir(fi) {
					if Debug {
						log.Printf("skipping directory %q under %s", fi.Name(), dir)
					}
					return filepath.SkipDir
				}
				return nil
			}
			if typ == os.ModeSymlink {
				base := filepath.Base(path)
				if strings.HasPrefix(base, ".#") {

					return nil
				}
				fi, err := os.Lstat(path)
				if err != nil {

					return nil
				}
				if shouldTraverse(dir, fi) {
					return traverseLink
				}
			}
			return nil
		}
		if err := fastWalk(srcDir, walkFn); err != nil {
			log.Printf("goimports: scanning directory %v: %v", srcDir, err)
		}
	}
}

func vendorlessImportPath(ipath string) string {

	if i := strings.LastIndex(ipath, "/vendor/"); i >= 0 {
		return ipath[i+len("/vendor/"):]
	}
	if strings.HasPrefix(ipath, "vendor/") {
		return ipath[len("vendor/"):]
	}
	return ipath
}

var loadExports func(expectPackage, dir string) map[string]bool = loadExportsGoPath

func loadExportsGoPath(expectPackage, dir string) map[string]bool {
	if Debug {
		log.Printf("loading exports in dir %s (seeking package %s)", dir, expectPackage)
	}
	exports := make(map[string]bool)

	ctx := build.Default

	ctx.ReadDir = func(dir string) (notTests []os.FileInfo, err error) {
		all, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		notTests = all[:0]
		for _, fi := range all {
			name := fi.Name()
			if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
				notTests = append(notTests, fi)
			}
		}
		return notTests, nil
	}

	files, err := ctx.ReadDir(dir)
	if err != nil {
		log.Print(err)
		return nil
	}

	fset := token.NewFileSet()

	for _, fi := range files {
		match, err := ctx.MatchFile(dir, fi.Name())
		if err != nil || !match {
			continue
		}
		fullFile := filepath.Join(dir, fi.Name())
		f, err := parser.ParseFile(fset, fullFile, nil, 0)
		if err != nil {
			if Debug {
				log.Printf("Parsing %s: %v", fullFile, err)
			}
			return nil
		}
		pkgName := f.Name.Name
		if pkgName == "documentation" {

			continue
		}
		if pkgName != expectPackage {
			if Debug {
				log.Printf("scan of dir %v is not expected package %v (actually %v)", dir, expectPackage, pkgName)
			}
			return nil
		}
		for name := range f.Scope.Objects {
			if ast.IsExported(name) {
				exports[name] = true
			}
		}
	}

	if Debug {
		exportList := make([]string, 0, len(exports))
		for k := range exports {
			exportList = append(exportList, k)
		}
		sort.Strings(exportList)
		log.Printf("loaded exports in dir %v (package %v): %v", dir, expectPackage, strings.Join(exportList, ", "))
	}
	return exports
}

var findImport func(pkgName string, symbols map[string]bool, filename string) (foundPkg string, rename bool, err error) = findImportGoPath

func findImportGoPath(pkgName string, symbols map[string]bool, filename string) (foundPkg string, rename bool, err error) {
	if inTests {
		testMu.RLock()
		defer testMu.RUnlock()
	}

	if pkg, rename, ok := findImportStdlib(pkgName, symbols); ok {
		return pkg, rename, nil
	}
	if pkgName == "rand" && symbols["Read"] {

		return "", false, nil
	}

	populateIgnoreOnce.Do(populateIgnore)

	scanGoRootOnce.Do(scanGoRoot) 
	if !fileInDir(filename, build.Default.GOROOT) {
		scanGoPathOnce.Do(scanGoPath) 
	}
	<-scanGoRootDone

	var candidates []*pkg
	for _, pkg := range dirScan {
		if pkgIsCandidate(filename, pkgName, pkg) {
			candidates = append(candidates, pkg)
		}
	}

	sort.Sort(byImportPathShortLength(candidates))
	if Debug {
		for i, pkg := range candidates {
			log.Printf("%s candidate %d/%d: %v", pkgName, i+1, len(candidates), pkg.importPathShort)
		}
	}

	done := make(chan struct{}) 
	defer close(done)

	rescv := make([]chan *pkg, len(candidates))
	for i := range candidates {
		rescv[i] = make(chan *pkg)
	}
	const maxConcurrentPackageImport = 4
	loadExportsSem := make(chan struct{}, maxConcurrentPackageImport)

	go func() {
		for i, pkg := range candidates {
			select {
			case loadExportsSem <- struct{}{}:
				select {
				case <-done:
				default:
				}
			case <-done:
				return
			}
			pkg := pkg
			resc := rescv[i]
			go func() {
				if inTests {
					testMu.RLock()
					defer testMu.RUnlock()
				}
				defer func() { <-loadExportsSem }()
				exports := loadExports(pkgName, pkg.dir)

				for symbol := range symbols {
					if !exports[symbol] {
						pkg = nil
						break
					}
				}
				select {
				case resc <- pkg:
				case <-done:
				}
			}()
		}
	}()
	for _, resc := range rescv {
		pkg := <-resc
		if pkg == nil {
			continue
		}

		needsRename := path.Base(pkg.importPath) != pkgName
		return pkg.importPathShort, needsRename, nil
	}
	return "", false, nil
}

func pkgIsCandidate(filename, pkgIdent string, pkg *pkg) bool {

	if !canUse(filename, pkg.dir) {
		return false
	}

	lastTwo := lastTwoComponents(pkg.importPathShort)
	if strings.Contains(lastTwo, pkgIdent) {
		return true
	}
	if hasHyphenOrUpperASCII(lastTwo) && !hasHyphenOrUpperASCII(pkgIdent) {
		lastTwo = lowerASCIIAndRemoveHyphen(lastTwo)
		if strings.Contains(lastTwo, pkgIdent) {
			return true
		}
	}

	return false
}

func hasHyphenOrUpperASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '-' || ('A' <= b && b <= 'Z') {
			return true
		}
	}
	return false
}

func lowerASCIIAndRemoveHyphen(s string) (ret string) {
	buf := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch {
		case b == '-':
			continue
		case 'A' <= b && b <= 'Z':
			buf = append(buf, b+('a'-'A'))
		default:
			buf = append(buf, b)
		}
	}
	return string(buf)
}

func canUse(filename, dir string) bool {

	if !strings.Contains(dir, "vendor") && !strings.Contains(dir, "internal") {
		return true
	}

	dirSlash := filepath.ToSlash(dir)
	if !strings.Contains(dirSlash, "/vendor/") && !strings.Contains(dirSlash, "/internal/") && !strings.HasSuffix(dirSlash, "/internal") {
		return true
	}

	absfile, err := filepath.Abs(filename)
	if err != nil {
		return false
	}
	absdir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absfile, absdir)
	if err != nil {
		return false
	}
	relSlash := filepath.ToSlash(rel)
	if i := strings.LastIndex(relSlash, "../"); i >= 0 {
		relSlash = relSlash[i+len("../"):]
	}
	return !strings.Contains(relSlash, "/vendor/") && !strings.Contains(relSlash, "/internal/") && !strings.HasSuffix(relSlash, "/internal")
}

func lastTwoComponents(v string) string {
	nslash := 0
	for i := len(v) - 1; i >= 0; i-- {
		if v[i] == '/' || v[i] == '\\' {
			nslash++
			if nslash == 2 {
				return v[i:]
			}
		}
	}
	return v
}

type visitFn func(node ast.Node) ast.Visitor

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	return fn(node)
}

func findImportStdlib(shortPkg string, symbols map[string]bool) (importPath string, rename, ok bool) {
	for symbol := range symbols {
		key := shortPkg + "." + symbol
		path := stdlib[key]
		if path == "" {
			if key == "rand.Read" {
				continue
			}
			return "", false, false
		}
		if importPath != "" && importPath != path {

			return "", false, false
		}
		importPath = path
	}
	if importPath == "" && shortPkg == "rand" && symbols["Read"] {
		return "crypto/rand", false, true
	}
	return importPath, false, importPath != ""
}

func fileInDir(file, dir string) bool {
	rest := strings.TrimPrefix(file, dir)
	if len(rest) == len(file) {

		return false
	}

	return len(rest) == 0 || rest[0] == '/' || rest[0] == '\\'
}
