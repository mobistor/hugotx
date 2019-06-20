package main

import (
	"encoding/json"
	"fmt"
	"github.com/bregydoc/gtranslate"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
  "time"
	"github.com/gernest/front"
  "bufio"
)

const SPACE = "  "
var PWD = ""

type replace struct {
	Name 			string 		`yaml:"name"`
  Value 		string 		`yaml:"value"`
}

type folder struct {
	TplFile 	string   	`yaml:"tplfile"`
	TplLang 	string   	`yaml:"tpllang"`
	DstPath		string   	`yaml:"dstpath"`
	DstExt		string    `yaml:"dstext"`
	LangSub 	bool     	`yaml:"langsub"`
	LangIdx 	bool     	`yaml:"langidx"`
	YamlFmt		string  	`yaml:"yamlfmt"`
	Skips   	[]string 	`yaml:"skips"`
  Replaces 	[]replace `yaml:"replaces"`
}

type conf struct {
	Languages []string 	`yaml:"languages"`
	LangList  folder  	`yaml:"langlist"`
	YAML   		[]folder 	`yaml:"yaml"`
  JSON			[]folder  `yaml:"json"`
}

type trans struct {
	output   *os.File
	tplFile  string
	dstFile  string
	fromLang string
	toLang   string
	skips    []string
  replaces []replace
}

// for i18n translation files
type Pair struct {
  Id string `yaml:"id"`
  Translation string `yaml:"translation"`
}


func TRANSLATE(text, fromLang, toLang string) string {

	c, err := gtranslate.Translate(text, language.Make(fromLang), language.Make(toLang))
	if err != nil {
		log.Fatalf("translate error: %v", err)
	}
	return c
}

func TRANSLATE_WITH_REPLACE (text string, tx *trans) string {
	s0 := TRANSLATE(text, tx.fromLang, tx.toLang)
	var s1, s2 = string(s0), ""
	for _, r := range tx.replaces {
		s2 = strings.ReplaceAll(s1, r.Name, r.Value)
		s1 = s2
	}
	return s1
}

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

//
// For json to js
//
func parseJsonStringMap(tx *trans, mp map[string]interface{}, space string, newline bool) {
	first := true
	for key, value := range mp {
		if !first {
			fmt.Fprintf(tx.output, ",\n")
		}
		if first {
			first = false
		}
		switch v := value.(type) {
		case string:
			text := strings.TrimSuffix(value.(string), "\n")
			//fmt.Fprintf(tx.output, "%s%s: '%s'\n", space, key, text)
			if !contains(tx.skips, key) {
				//fmt.Fprintf(tx.output, " >\n%s  %v\n", space, TRANSLATE_WITH_REPLACE(text, tx))
				fmt.Fprintf(tx.output, "%s%s: '%s'", space, key, TRANSLATE_WITH_REPLACE(text, tx))
			} else {
				//fmt.Fprintf(tx.output, "%s\n", text)
				fmt.Fprintf(tx.output, "%s%s: '%s'", space, key, text)
			}
		case bool:
			fmt.Fprintf(tx.output, "%v\n", v)
		case int:
			fmt.Fprintf(tx.output, "%v\n", v)
		case float64:
			fmt.Fprintf(tx.output, "%v\n", v)
		case map[interface{}]interface{}:
			parseMap(tx, v, space+SPACE, true)
		case []interface{}:
			fmt.Fprintf(tx.output, "\n")
			parseArray(tx, v, space+SPACE, key)
		case map[string]interface{}:
			fmt.Fprintf(tx.output, "%s%s: {\n", space, key)
			parseJsonStringMap(tx, v, space+SPACE, true)
			fmt.Fprintf(tx.output, "%s}", space)
		default:
			log.Fatalf("# unknow value %+v\n", value)
		}
	}
	fmt.Fprintf(tx.output, "\n")
}

//
// For yaml by "github.com/gernest/front", front matter with content
//
func parseStringMap(tx *trans, mp map[string]interface{}, space string, newline bool) {
	done := false
	for key, value := range mp {
		if !done {
			if newline {
				fmt.Fprintf(tx.output, "\n%s", space)
			}
			done = true
		} else {
			fmt.Fprintf(tx.output, "%s", space)
		}
		fmt.Fprintf(tx.output, "%s: ", key)

		switch v := value.(type) {
		case string:
			text := strings.TrimSuffix(value.(string), "\n")
			if !contains(tx.skips, key) {
				fmt.Fprintf(tx.output, " >\n%s  %v\n", space, TRANSLATE_WITH_REPLACE(text, tx))
			} else {
				fmt.Fprintf(tx.output, "%s\n", text)
			}
		case bool:
			fmt.Fprintf(tx.output, "%v\n", v)
		case int:
			fmt.Fprintf(tx.output, "%v\n", v)
		case float64:
			fmt.Fprintf(tx.output, "%v\n", v)
		case map[interface{}]interface{}:
			parseMap(tx, v, space+SPACE, true)
		case []interface{}:
			fmt.Fprintf(tx.output, "\n")
			parseArray(tx, v, space+SPACE, key)
		default:
			log.Fatalf("# unknow value %+v\n", value)
		}
	}
}

//
// For yaml by "gopkg.in/yaml.v2"
//
func parseMap(tx *trans, mp map[interface{}]interface{}, space string, newline bool) {
	done := false
	for key, value := range mp {
		if !done {
			if newline {
				fmt.Fprintf(tx.output, "\n%s", space)
			}
			done = true
		} else {
			fmt.Fprintf(tx.output, "%s", space)
		}
		switch k := key.(type) {
		case string:
			fmt.Fprintf(tx.output, "%v: ", k)
		default:
			log.Fatalf("# map key unknown: %+v", k)
		}
		switch v := value.(type) {
		case string:
			text := strings.TrimSuffix(value.(string), "\n")
			if !contains(tx.skips, key.(string)) {
				fmt.Fprintf(tx.output, " >\n%s  %v\n", space, TRANSLATE_WITH_REPLACE(text, tx))
			} else {
				fmt.Fprintf(tx.output, "%s\n", text)
			}
		case bool:
			fmt.Fprintf(tx.output, "%v\n", v)
		case int:
			fmt.Fprintf(tx.output, "%v\n", v)
		case float64:
			fmt.Fprintf(tx.output, "%v\n", v)
		case map[interface{}]interface{}:
			parseMap(tx, v, space+SPACE, true)
		case []interface{}:
			fmt.Fprintf(tx.output, "\n")
			parseArray(tx, v, space+SPACE, key.(string))
		default:
			log.Fatalf("# unknow value %+v\n", value, key)
		}
	}
}

func parseArray(tx *trans, arr []interface{}, space string, key string ) {
	for _, value := range arr {
		switch v := value.(type) {
		case string:
			text := strings.TrimSuffix(value.(string), "\n")
			if !contains(tx.skips, key) {
				fmt.Fprintf(tx.output, "%s- %v\n", space, TRANSLATE_WITH_REPLACE(text, tx))
			} else {
				fmt.Fprintf(tx.output, "%s- %s\n", space, text)
			}
		case map[interface{}]interface{}:
			fmt.Fprintf(tx.output, "%s- ", space)
			parseMap(tx, v, space+"  ", false)
		default:
			log.Fatalf("# unknow array %+v\n", v)
		}
	}
}

func file_is_exists(f string) error {
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		return err
	}
	return nil
}

func dir_must_exist(d string) error {
	if err :=	os.MkdirAll(d, os.ModeDir | 0777); err != nil {
		log.Fatalf("Fail to create folder of $d !", d)
	}
	return nil
}

func no_update(src_file, dst_file string) (bool) {
	src, err := os.Stat(src_file)
	if err != nil {
		return false
	}
  dst, err := os.Stat(dst_file)
  if err != nil {
		return false
  }
	src_time := src.ModTime()
  dst_time := dst.ModTime()
  diff := src_time.Sub(dst_time)
  if diff < (time.Duration(0) * time.Second) {
		return true
  }
  return false
}


func load_md(tplFile string)(map[string]interface{}, string,error) {

	tpl, err := os.Open(tplFile)
  if err != nil {
    return nil, "", err
  }
	defer tpl.Close()

	m := front.NewMatter()
	m.Handle("---", front.YAMLHandler)
	return m.Parse(bufio.NewReader(tpl))
}

func MD(tx *trans)(bool, error) {

  // load and parse .md file
	fm, body, err := load_md(tx.tplFile)
	if err != nil {
		log.Fatalf("Fail to load and parse %s, please check it!\n", tx.tplFile)
	}

	// create output file
	tx.output, err = os.Create(tx.dstFile)
	if err != nil {
		log.Fatalf("Fail to create target file of %s !\n", tx.dstFile)
	}
	defer tx.output.Close()

	//fmt.Printf("The front matter is:\n%#v\n", fm)
	//fmt.Printf("The body is:\n%q\n", body)

	// handle front_matter part
	fmt.Fprintf(tx.output, "---\n")
	parseStringMap(tx, fm, "", false)
	fmt.Fprintf(tx.output, "---\n")

	// handle body part
	fmt.Fprintf(tx.output, "%s", TRANSLATE_WITH_REPLACE(body, tx))
  return true, nil
}


func load_i18n(tplFile string)([]Pair, error) {

	tpl, err := ioutil.ReadFile(tplFile)
  if err != nil {
    return nil, err
  }

  pairs := make([]Pair,0)
  err = yaml.Unmarshal(tpl, &pairs)
  if err != nil {
    fmt.Println(err.Error())
    return nil, err
  }
  return pairs, nil
}

func I18N(tx *trans)(bool, error) {

  pairs, err := load_i18n(tx.tplFile)
	if err != nil {
		log.Fatalf("Fail to load and parse %s, please check it!\n", tx.tplFile)
	}

  // create output file
	tx.output, err = os.Create(tx.dstFile)
	if err != nil {
		log.Fatalf("Fail to create target file of %s !\n", tx.dstFile)
	}
	defer tx.output.Close()

	for _, item := range pairs {
		fmt.Fprintf(tx.output, "- id: %s\n  translation: %s\n\n", item.Id, TRANSLATE_WITH_REPLACE(item.Translation, tx))
	}
  return true, nil
}

func load_yaml(tplFile string)(map[interface{}]interface{}, error) {

	tpl, err := ioutil.ReadFile(tplFile)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	mp := make(map[interface{}]interface{})
	err = yaml.Unmarshal(tpl, mp)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return mp, nil
}

func YAML(tx *trans)(bool, error) {

	// load and parse .md file
	fm, err := load_yaml(tx.tplFile)
	if err != nil {
		log.Fatalf("Fail to load and parse %s, please check it!\n", tx.tplFile)
	}

	// create output file
	tx.output, err = os.Create(tx.dstFile)
	if err != nil {
		log.Fatalf("Fail to create target file of %s !\n", tx.dstFile)
	}
	defer tx.output.Close()
	parseMap(tx, fm, "", false)

	return true, nil
}

func (c *conf) produceYAML() {

	for _, folder := range c.YAML {

		tplFile := PWD + "/" + folder.TplFile
    if err := file_is_exists(tplFile); err != nil {
    		log.Printf("template of %s not exists\n", tplFile)
    		continue
    }

    dstPath := PWD + "/" + folder.DstPath
    dstPath = strings.TrimSuffix(dstPath, "/")
    if err := dir_must_exist(dstPath); err != nil {
    		log.Printf("Fail to create target folder of %s !\n", dstPath)
    		continue
    }

    _, file := path.Split(tplFile)
    // parse file=base.ext
    ext := filepath.Ext(file)
    base := strings.TrimSuffix(file, ext)
    // fmt.Println("tplFile=", tplFile)
    // fmt.Println("dstPath=", dstPath)

    UPDATE := false
    fmt.Printf("analyzing %s ...", file)

		for _, lang := range c.Languages {
			target := ""
    	// produce correct target path
    	if folder.LangSub {
				target = dstPath + "/" + lang
				dir_must_exist(target)
				target = target + "/" + base + folder.DstExt
  		} else {
    		if folder.LangIdx {
    			target = dstPath + "/" + lang + folder.DstExt
     		} else {
     			target = dstPath + "/" + base + "." + lang + folder.DstExt
     		}
  		}
  		// fmt.Println("target=", target)

  		// check if target is up to date, compared to tplFile
			if no_update(tplFile, target) {
		    continue
		  }

  		tx := trans{tplFile: tplFile, dstFile: target, fromLang: folder.TplLang, toLang: lang, skips: folder.Skips, replaces: folder.Replaces}

      var update = false
      var err error
      switch folder.YamlFmt {
      	case "yaml":
      		update, err = YAML(&tx)
      	case "i18n":
      		update, err = I18N(&tx)
      	case "md":
      		update, err = MD(&tx)
      	default:
      		continue
      }
      if update && err == nil {
      	if rel, err := filepath.Rel(PWD, target); err == nil {
        	fmt.Printf("\n  %s generated.", rel)
        } else {
        	_, file := path.Split(target)
        	fmt.Printf("\n  %s generated.", file)
        }
      }
      UPDATE = UPDATE || update
		}
		if !UPDATE {
			fmt.Printf("(no update)\n")
		} else {
			fmt.Printf("\n")
		}

	}
}


func (c *conf) langlist()(bool, error) {

	dstPath := PWD + "/" + c.LangList.DstPath
  dstPath = strings.TrimSuffix(dstPath, "/")
  if err := dir_must_exist(dstPath); err != nil {
  	log.Fatalf("Fail to create language list path in %s !\n", dstPath)
  }
  ext := filepath.Ext(c.LangList.TplFile)
  base := strings.TrimSuffix(c.LangList.TplFile, ext)

	for _, lang := range c.Languages {

		target := ""
    // produce correct target path
    if c.LangList.LangSub {
			target = dstPath + "/" + lang
			dir_must_exist(target)
			target = target + "/" + base + c.LangList.DstExt
  	} else {
    	if c.LangList.LangIdx {
    		target = dstPath + "/" + lang + c.LangList.DstExt
     	} else {
     		target = dstPath + "/" + base + "." + lang + c.LangList.DstExt
     	}
  	}

		/* datapath := PWD + "/" + c.LangList
		name := display.Tags(language.Make(lang)) //language.Make(lang))
		target := datapath + "/" + lang
		os.MkdirAll(target, 0777)

		target = target + "/languages.yaml" */

		if rel, err := filepath.Rel(PWD, target); err == nil {
			fmt.Printf("analyzing %s ...", rel)
    } else {
      _, file := path.Split(target)
      fmt.Printf("analyzing %s ...", file)
    }

		if no_update("./txconf.yaml", target) {
			fmt.Printf("(no update)\n")
			continue
  	}

		f, err := os.Create(target)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		defer f.Close()

		name := display.Tags(language.Make(lang))
		for _, t := range c.Languages {
			x := language.Make(t)
			switch c.LangList.YamlFmt {
				case "self":
					fmt.Fprintf(f, "%s: %s\n", t, name.Name(x))
				case "global":
					fmt.Fprintf(f, "%s: %s\n", t, display.Self.Name(x))
    	}
		}
		fmt.Printf("generated.\n")
	}
	return true, nil
}

func load_json(tplFile string)(map[string]interface{}, error) {

	tpl, err := ioutil.ReadFile(tplFile)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	mp := make(map[string]interface{})
	err = json.Unmarshal(tpl, &mp)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return mp, nil
}

func JSON(tx *trans)(bool, error) {

	// load and parse .md file
	mp, err := load_json(tx.tplFile)
	if err != nil {
		log.Fatalf("Fail to load and parse %s, please check it!\n", tx.tplFile)
	}

	// create output file
	/* tx.output, err = os.Create(tx.dstFile)
	if err != nil {
		log.Fatalf("Fail to create target file of %s !\n", tx.dstFile)
	}
	defer tx.output.Close()*/
	tx.output = os.Stdout
	fmt.Fprintf(tx.output, "\nexport default {\n")
	parseJsonStringMap(tx, mp, SPACE, false)
	fmt.Fprintf(tx.output, "\n}")

	return true, nil
}

func (c *conf) produceJSON() {

	for _, folder := range c.JSON {

		tplFile := PWD + "/" + folder.TplFile
    if err := file_is_exists(tplFile); err != nil {
    		log.Printf("template of %s not exists\n", tplFile)
    		continue
    }

    dstPath := PWD + "/" + folder.DstPath
    dstPath = strings.TrimSuffix(dstPath, "/")
    if err := dir_must_exist(dstPath); err != nil {
    		log.Printf("Fail to create target folder of %s !\n", dstPath)
    		continue
    }

    _, file := path.Split(tplFile)
    // parse file=base.ext
    ext := filepath.Ext(file)
    base := strings.TrimSuffix(file, ext)
    // fmt.Println("tplFile=", tplFile)
    // fmt.Println("dstPath=", dstPath)

    UPDATE := false
    fmt.Printf("analyzing %s ...", file)

		for _, lang := range c.Languages {
			target := ""
    	// produce correct target path
    	if folder.LangSub {
				target = dstPath + "/" + lang
				dir_must_exist(target)
				target = target + "/" + base + folder.DstExt
  		} else {
    		if folder.LangIdx {
    			target = dstPath + "/" + lang + folder.DstExt
     		} else {
     			target = dstPath + "/" + base + "." + lang + folder.DstExt
     		}
  		}
  		// fmt.Println("target=", target)

  		// check if target is up to date, compared to tplFile
			if no_update(tplFile, target) {
		    continue
		  }

  		tx := trans{tplFile: tplFile, dstFile: target, fromLang: folder.TplLang, toLang: lang, skips: folder.Skips, replaces: folder.Replaces}
      update, err := JSON(&tx)
      if update && err == nil {
      	if rel, err := filepath.Rel(PWD, target); err == nil {
        	fmt.Printf("\n  %s generated.", rel)
        } else {
        	_, file := path.Split(target)
        	fmt.Printf("\n  %s generated.", file)
        }
      }
      UPDATE = UPDATE || update
		}
		if !UPDATE {
			fmt.Printf("(no update)\n")
		} else {
			fmt.Printf("\n")
		}

	}
}


func (c *conf) getConf() *conf {

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("%s!", err.Error())
	}
	PWD = pwd

	conf_path := PWD + "/txconf.yaml"
	if err = file_is_exists(conf_path); err != nil {
		log.Fatalf("./txconf.yaml not found!")
	}

	yamlFile, err := ioutil.ReadFile(conf_path)
	if err != nil {
		log.Fatalf("loading ./txconf.yaml fail!")
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("parsing ./txconf.yaml fail due to %s\n", err.Error())
	}

	return c
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var c conf
	c.getConf()
  c.produceJSON()

	// c.langlist()
	// c.produceYAML()
}
