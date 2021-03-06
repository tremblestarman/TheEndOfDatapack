package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-ego/gse"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var db *gorm.DB
var seg gse.Segmenter

const (
	orderByPostTime       = "post_time"
	orderByPostTimeDesc   = "post_time desc"
	orderByUpdateTime     = "update_time"
	orderByUpdateTimeDesc = "update_time desc"
	datapackPageCount     = 15
	tagPageCount          = 100
	maxDatapackShownInTag = 100
	authorPageCount       = 24
	keyWordHighlightHead  = "<span class=\"highlight\">"
	keyWordHighlightTail  = "</span>"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

type Auth struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}
type SearchResult struct {
	KeyWordCount int `json:"-"`
}

type Tag struct {
	SearchResult
	ID            string     `json:"-" gorm:"primary_key:true"`
	Tag           string     `json:"tag_name"`
	DefaultLang   string     `json:"tag_language"`
	DefaultLangId string     `json:"-"`
	DefaultTag    string     `json:"tag_default_name"`
	Type          int        `json:"tag_type"`
	Thumb         int        `json:"-"`
	Quotation     int        `json:"quotation"`
	Datapacks     []Datapack `json:"datapacks" gorm:"many2many:datapack_tags;association_foreignkey:id;foreignkey:id;association_jointable_foreignkey:datapack_id;jointable_foreignkey:tag_id;"`
}
type Author struct {
	SearchResult
	ID             string     `json:"-" gorm:"primary_key:true"`
	AuthorUid      string     `json:"-"`
	AuthorName     string     `json:"author_name"`
	Avatar         string     `json:"author_avatar"`
	Thumb          int        `json:"-"`
	Datapacks      []Datapack `json:"datapacks" gorm:"ForeignKey:AuthorID;AssociationForeignKey:ID"`
	RelatedAuthors []Author   `json:"-"`
}
type Datapack struct {
	SearchResult
	ID               string     `json:"-" gorm:"primary_key:true"`
	Link             string     `json:"datapack_link"`
	Name             string     `json:"datapack_name"`
	Author           Author     `json:"author"`
	AuthorID         string     `json:"-"`
	DefaultLang      string     `json:"datapack_language"`
	DefaultLangId    string     `json:"-"`
	DefaultName      string     `json:"datapack_default_name"`
	Source           string     `json:"source"`
	Intro            string     `json:"introduction"`
	FullContent      string     `json:"-"`
	PostTime         time.Time  `json:"-"`
	PostTimeString   string     `json:"post_time"`
	UpdateTime       time.Time  `json:"-"`
	UpdateTimeString string     `json:"update_time"`
	CoverExists      bool       `json:"-"`
	Thumb            int        `json:"-"`
	Tags             []Tag      `json:"tag" gorm:"many2many:datapack_tags;association_foreignkey:id;foreignkey:id;association_jointable_foreignkey:tag_id;jointable_foreignkey:datapack_id;"`
	RelatedDatapacks []Datapack `json:"-"`
}

func (d *Datapack) Initialize() *Datapack {
	d.PostTimeString = d.PostTime.Format("2006-01-02 15:04:05")
	d.UpdateTimeString = d.UpdateTime.Format("2006-01-02 15:04:05")
	d.Intro = "    " + strings.ReplaceAll(d.Intro, "\n", ".\n    ") + "."
	d.Intro = strings.ReplaceAll(d.Intro, "<", "＜") // '<' and '>', which confuse function 'unescape'
	d.Intro = strings.ReplaceAll(d.Intro, ">", "＞") // replace '<>' with unicode '＜＞'
	d.CoverExists, _ = PathExists("bin/img/cover/" + d.ID + ".png")
	return d
}
func (d *Datapack) GetTags(tag string) *Datapack {
	rows, _ := db.Raw("select tags.*, tags." + tag + " as tag from (select datapack_tags.tag_id as id from datapack_tags where datapack_id = '" + d.ID + "') as ids left join tags on tags.id = ids.id order by tags.type, tags.default_tag, tags.default_tag DESC;").Rows()
	for rows.Next() {
		var tag Tag
		_ = db.ScanRows(rows, &tag)
		d.Tags = append(d.Tags, tag)
	}
	_ = rows.Close()
	return d
}
func (d *Datapack) GetAuthor() *Datapack {
	rows, _ := db.Raw("select authors.* from authors where authors.id = '" + d.AuthorID + "'").Rows()
	for rows.Next() {
		_ = db.ScanRows(rows, &d.Author)
	}
	_ = rows.Close()
	return d
}
func (d *Datapack) GetRelated(language string) *Datapack {
	// Get Related Datapacks
	var sql = db
	var related DatapackRelated
	sql.Model(&DatapackRelated{}).
		Where("datapack_id = ?", d.ID).
		First(&related)
	if related.Related != "" {
		// Set language
		name, tag := "default_name", "default_tag"
		if language != "" && language != "default" {
			name, tag = "name_"+language, "tag_"+language
		}
		fmt.Println(related.GetTuple())
		sql.Model(&Datapack{}).
			Select("distinct datapacks.*, datapacks."+name+" as name"). // Set Datapack Name
			Preload("Tags", func(db *gorm.DB) *gorm.DB {                // Preload Tags
				return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag DESC") // Set Tag Name & Set Order
			}).
			Preload("Author").                                                       // Preload Author
			Where("datapacks.id IN " + related.GetTuple()).Find(&d.RelatedDatapacks) // Get All Related Datapacks
	}
	return d
}
func (a *Author) GetRelated(language string) {
	// Get Related Authors
	var sql = db
	var related AuthorRelated
	sql.Model(&AuthorRelated{}).
		Where("author_id = ?", a.ID).
		First(&related)
	if related.Related != "" {
		// Set language
		name, tag := "default_name", "default_tag"
		if language != "" && language != "default" {
			name, tag = "name_"+language, "tag_"+language
		}
		sql.Preload("Datapacks", func(db *gorm.DB) *gorm.DB { // Preload Datapacks
			return db.Select("*, datapacks." + name + " as name").Order("datapacks.post_time DESC") // Set Datapack Name & Set Order
		}).
			Preload("Datapacks.Tags", func(db *gorm.DB) *gorm.DB { // Preload Datapacks.Tags
				return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag DESC") // Set Tag Name & Set Order
			}).
			Where("authors.id IN " + related.GetTuple()).Find(&a.RelatedAuthors) // Get All Related Authors
	}
}
func KeyWordHighlight(raw *string, keywordsReg *string) {
	re := regexp.MustCompile("(?i)" + *keywordsReg)
	*raw = re.ReplaceAllStringFunc(*raw, func(s string) string {
		return keyWordHighlightHead + strings.ToUpper(s) + keyWordHighlightTail
	})
}

type IDRelated struct {
	Related     string
	RelatedList []string
}
type DatapackRelated struct {
	IDRelated
	DatapackID string
}
type AuthorRelated struct {
	IDRelated
	AuthorID string
}

func (DatapackRelated) TableName() string {
	return "datapacks_related"
}
func (AuthorRelated) TableName() string {
	return "authors_related"
}
func (i *IDRelated) GetTuple() string {
	if i.Related == "" {
		return "('')"
	}
	r := strings.ReplaceAll(i.Related, ";", "','")
	return "('" + r[0:len(r)-2] + ")"
}

// Connect
func Connect() {
	var auth Auth
	jstring, err := ioutil.ReadFile("util/auth.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(jstring, &auth)
	if err != nil {
		panic(err)
	}
	dsn := auth.User + ":" + auth.Password + "@tcp(" + auth.Host + ":" + strconv.Itoa(auth.Port) + ")/datapack_collection?charset=UTF8MB4&parseTime=True&loc=Local"
	fmt.Println(dsn + " connecting..")
	db, err = gorm.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err)
	}
}
func GetLanguages() *map[string]interface{} {
	var languages map[string]interface{}
	jstring, err := ioutil.ReadFile("util/languages.json")
	if err != nil {
		fmt.Println("languages.json: ", err)
	}
	err = json.Unmarshal(jstring, &languages)
	if err != nil {
		fmt.Println("languages.json -> map[string] err: ", err)
	}
	return &languages
}

// List
func dateRange(sql *gorm.DB, dateRange int, col string) *gorm.DB {
	switch dateRange {
	case 1: // Today
		sql = sql.Where("TO_DAYS(" + col + ") = TO_DAYS(NOW())")
	case 2: // 3 Days
		sql = sql.Where("TO_DAYS(NOW()) - TO_DAYS(" + col + ") <= 3")
	case 3: // 7 Days
		sql = sql.Where("DATE_SUB(CURDATE(), INTERVAL 7 DAY) <= date(" + col + ")")
	case 4: // 1 Month
		sql = sql.Where("DATE_SUB(CURDATE(), INTERVAL 30 DAY) <= date(" + col + ")")
	case 5: // this Month
		sql = sql.Where("DATE_FORMAT(" + col + ", '%Y%m') = DATE_FORMAT(CURDATE(), '%Y%m')")
	case 6: // this Year
		sql = sql.Where("YEAR(" + col + ")=YEAR(NOW())")
	}
	return sql
}
func datapackFilter(sql *gorm.DB, source string, version string, postTimeRange int, updateTimeRange int) (*gorm.DB, string) {
	table := "datapacks"
	if source != "" {
		sql = sql.Where("source = '" + source + "'")
	}
	if version != "" {
		table = "(select dt.datapack_id as id from tags left join datapack_tags as dt on tags.id = dt.tag_id where tags.type = 1 and tags.default_tag = '" + version + "') as vs left join datapacks on vs.id = datapacks.id"
	}
	if postTimeRange != 0 {
		sql = dateRange(sql, postTimeRange, "datapacks.post_time")
	}
	if updateTimeRange != 0 {
		sql = dateRange(sql, updateTimeRange, "datapacks.update_time")
	}
	return sql, table
}
func ListDatapacks(language string, page int, order string, source string, version string, postTimeRange int, updateTimeRange int) (*[]Datapack, int) {
	var datapacks []Datapack
	if page < 1 {
		page = 1
	}
	var offset, limit = (page - 1) * datapackPageCount, datapackPageCount
	var sql = db
	// Set language
	name, tag := "default_name", "default_tag"
	if language != "" && language != "default" {
		name, tag = "name_"+language, "tag_"+language
	}
	// Query
	total := 0
	sql = db.Model(&Datapack{}).
		Select("distinct datapacks.*, datapacks."+name+" as name"). // Set Datapack Name
		Preload("Tags", func(db *gorm.DB) *gorm.DB {                // Preload Tags
			return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag, tags.default_tag DESC") // Set Tag Name & Set Order
		}).
		Preload("Author") // Preload Author
		// Preload is quicker than Rows Iterator, because Iterator use extra codes to reflect data on structure
	sql, table := datapackFilter(sql, source, version, postTimeRange, updateTimeRange)       // Filter
	sql.Table(table).Count(&total).Order(order).Offset(offset).Limit(limit).Find(&datapacks) // Count All & Only Find Datapack to be Shown
	// Initialize Datapacks
	for i := 0; i < len(datapacks); i++ {
		datapacks[i].Initialize()
	}
	return &datapacks, total
}
func GetDatapack(language string, id string) *Datapack {
	var datapacks []Datapack
	var sql = db
	// Set language
	name, tag := "default_name", "default_tag"
	if language != "" && language != "default" {
		name, tag = "name_"+language, "tag_"+language
	}
	sql.Model(&Datapack{}).
		Select("distinct datapacks.*, datapacks."+name+" as name"). // Set Datapack Name
		Preload("Tags", func(db *gorm.DB) *gorm.DB {                // Preload Tags
			return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag DESC") // Set Tag Name & Set Order
		}).
		Preload("Author"). // Preload Author
		Where("datapacks.id = '" + id + "'").First(&datapacks)
	if len(datapacks) > 0 {
		datapacks[0].Initialize()
		datapacks[0].GetRelated(language)
		return &(datapacks[0])
	}
	return nil
}
func RemoveDuplicates(a []string) (ret []string) {
	sort.Strings(a)
	aLen := len(a)
	for i := 0; i < aLen; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 || a[i] == " " || a[i] == "," || a[i] == "，" {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}
func ListTags(language string, page int, tag string) (*[]Tag, int) {
	var tags []Tag
	if page < 1 {
		page = 1
	}
	var offset, limit = (page - 1) * tagPageCount, tagPageCount
	// Set language
	_tag := "default_tag"
	if language != "" && language != "default" {
		_tag = "tag_" + language
	}
	var sql = db.Model(Tag{}).Select("distinct tags.*, tags." + _tag + " as tag").Order("tags.quotation DESC")
	// Return All
	total := 0
	if tag == "" {
		sql.Count(&total).Order("tags." + _tag + "").Offset(offset).Limit(limit).Find(&tags)
	} else { // Find via Reg
		sql.Where("tags." + _tag + " like '" + tag + "%'").Count(&total).Order("tags." + _tag + "").Offset(offset).Limit(limit).Find(&tags)
		for i := 0; i < len(tags); i++ { // Count and mark keywords
			KeyWordHighlight(&tags[i].Tag, &tag)
		}
	}
	return &tags, total
}
func (t Tag) GetSynonymousTag(language string) *[]Tag {
	var tags []Tag
	// Set language
	_tag := "default_tag"
	if language != "" && language != "default" {
		_tag = "tag_" + language
	}
	db.Select("distinct tags.*, tags.default_tag as tag").
		Where("tags."+_tag+" = '"+t.Tag+"'").Not("tags.id = ?", t.ID).Find(&tags)
	return &tags
}
func GetTag(page int, language string, id string) (*Tag, int) {
	var tags []Tag
	var sql = db
	// Set language
	name, tag := "default_name", "default_tag"
	if language != "" && language != "default" {
		name, tag = "name_"+language, "tag_"+language
	}
	sql.Model(&Tag{}).
		Where("tags.id = '" + id + "'"). // Find Tag Id
		First(&tags)                     // FindOne
	if len(tags) > 0 && tags[0].Type < 2 { // Got Tag of Source or Version
		return &(tags[0]), -1
	}
	// Query
	if page < 1 {
		page = 1
	}
	var offset, limit = (page - 1) * maxDatapackShownInTag, maxDatapackShownInTag
	sql.Model(&Tag{}).
		Select("distinct tags.*, tags."+tag+" as tag").   // Set Tag in desired language
		Preload("Datapacks", func(db *gorm.DB) *gorm.DB { // Preload Datapacks
			return db.Select("*, datapacks." + name + " as name").Order("datapacks.post_time DESC").Offset(offset).Limit(limit) // Set Datapack Name & Set Order
		}).
		Preload("Datapacks.Tags", func(db *gorm.DB) *gorm.DB { // Preload Datapacks.Tags
			return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag DESC") // Set Tag Name & Set Order
		}).
		Where("tags.id = '" + id + "'"). // Find Tag Id
		First(&tags)                     // Find One
	if len(tags) > 0 {
		return &(tags[0]), tags[0].Quotation
	}
	return nil, 0
}
func ListAuthors(page int, author string) (*[]Author, int) {
	var authors []Author
	if page < 1 {
		page = 1
	}
	var offset, limit = (page - 1) * authorPageCount, authorPageCount
	var sql = db
	// Return All
	total := 0
	if author == "" {
		sql.Model(Author{}).Select("distinct authors.*").Count(&total).Order("authors.author_name").Offset(offset).Limit(limit).Find(&authors)
	} else { // Find via Reg
		sql.Model(Author{}).Where("authors.author_name like '" + author + "%'").Count(&total).Order("authors.author_name").Offset(offset).Limit(limit).Find(&authors)
		for i := 0; i < len(authors); i++ { // Count and mark keywords
			KeyWordHighlight(&authors[i].AuthorName, &author)
		}
	}
	return &authors, total
}
func GetAuthor(language string, id string) *Author {
	var authors []Author
	var sql = db
	// Set language
	name, tag := "default_name", "default_tag"
	if language != "" && language != "default" {
		name, tag = "name_"+language, "tag_"+language
	}
	// Query
	sql.Preload("Datapacks", func(db *gorm.DB) *gorm.DB { // Preload Datapacks
		return db.Select("*, datapacks." + name + " as name").Order("datapacks.post_time DESC") // Set Datapack Name & Set Order
	}).
		Preload("Datapacks.Tags", func(db *gorm.DB) *gorm.DB { // Preload Datapacks.Tags
			return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag DESC") // Set Tag Name & Set Order
		}).
		Where("authors.id = '" + id + "'"). // Find Tag Id
		First(&authors)                     // Find One
	if len(authors) > 0 {
		authors[0].GetRelated(language)
		return &(authors[0])
	}
	return nil
}

// Search
func getKeywords(text string) (*string, []string) {
	hmm := seg.CutSearch(text, true)
	var words []string
	words = RemoveDuplicates(hmm)
	keywordsReg := strings.Join(words, "|")
	return &keywordsReg, words
}
func unionByWords(words []string, count string, wordSelect func(word string) string, postPosition string) string {
	var selects []string
	for _, word := range words {
		selects = append(selects, wordSelect(word))
	}
	sql := "select id, " + count + " as count from (" + strings.Join(selects, " union all ") + ") " + postPosition + " "
	return sql
}
func SearchDatapacks(language string, page int, content string, source string, version string, postTimeRange int, updateTimeRange int) (*[]Datapack, int) {
	if page < 1 {
		page = 1
	}
	var datapacks []Datapack
	// search via datapacks.name or datapacks.name_zh or datapacks.content or t.tag (type = 3 or 4
	var sql = db
	var offset, limit = (page - 1) * datapackPageCount, datapackPageCount
	keywordsReg, keywords := getKeywords(content)
	// Set language
	name, tag := "default_name", "default_tag"
	if language != "" && language != "default" {
		name, tag = "name_"+language, "tag_"+language
	}
	// Query
	total := 0
	sql = db.Model(&Datapack{}).
		Select("distinct datapacks.*, datapacks."+name+" as name, count as key_word_count"). // Set Datapack Name
		Preload("Tags", func(db *gorm.DB) *gorm.DB {                                         // Preload Tags
			return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag DESC") // Set Tag Name & Set Order
		}).
		Preload("Author") // Preload Author
	sql, table := datapackFilter(sql, source, version, postTimeRange, updateTimeRange) // Filter, Using Joined Table
	// Set Search Expression
	table = `(select id, sum(count) as count from (` + unionByWords(keywords, "sum(count)", func(word string) string {
		return "select id, count as count from datapacks_" + name + "_ii where word like '" + word + "%'"
	}, "as _name group by id") + ` union all ` + unionByWords(keywords, "sum(count)", func(word string) string {
		return "select id, count as count from datapacks_intro_ii where word like '" + word + "%'"
	}, "as _intro group by id") + ` union all ` + unionByWords(keywords, "1", func(word string) string {
		return "select dt.datapack_id as id from tags left join datapack_tags as dt on tags.id = dt.tag_id where tags.default_tag like '" + word + "%' and tags.type >= 2"
	}, "as _intro group by id") + `) as results group by id 
        ) as ids left join ` + table + ` on ids.id = datapacks.id`
	sql.Table(table).Where("datapacks.id is not NULL").Count(&total).Order("count DESC").Offset(offset).Limit(limit).Find(&datapacks) // Find All
	// Count and Mark Keywords
	for i := 0; i < len(datapacks); i++ {
		datapacks[i].Initialize()
		KeyWordHighlight(&datapacks[i].Name, keywordsReg)
		KeyWordHighlight(&datapacks[i].Intro, keywordsReg)
		for _, t := range datapacks[i].Tags {
			if t.Type == 3 || t.Type == 4 {
				KeyWordHighlight(&t.Tag, keywordsReg)
			}
		}
	}
	return &datapacks, total
}
func AccurateSearchDatapacks(language string, page int, name string, intro string, author string, source string, version string, postTimeRange int, updateTimeRange int) (*[]Datapack, int) {
	if page < 1 {
		page = 1
	}
	var datapacks []Datapack
	// search via datapacks.name or name_zh and datapacks.intro and authors.author_name and date
	var sql = db
	var offset, limit = (page - 1) * datapackPageCount, datapackPageCount
	var keywordsMatrix [3]*string
	selects := make([]string, 0)
	// Set language
	_name, tag := "default_name", "default_tag"
	if language != "" && language != "default" {
		_name, tag = "name_"+language, "tag_"+language
	}
	// Query
	total := 0
	sql = db.Model(&Datapack{}).
		Select("distinct datapacks.*, datapacks."+_name+" as name, count as key_word_count"). // Set Datapack Name
		Preload("Tags", func(db *gorm.DB) *gorm.DB {                                          // Preload Tags
			return db.Select("*, tags." + tag + " as tag").Order("tags.type, tags.default_tag DESC") // Set Tag Name & Set Order
		}).
		Preload("Author"). // Preload Authors
		Joins("JOIN authors ON datapacks.author_id = authors.id")
	// Query Name
	if name != "" {
		var keywords []string
		keywordsMatrix[0], keywords = getKeywords(name)
		selects = append(selects, unionByWords(keywords, "sum(count)", func(word string) string {
			return "select id, count as count from datapacks_" + _name + "_ii where word like '" + word + "%'"
		}, "as _name group by id having count(*) >= "+strconv.Itoa(len(keywords))))
	}
	// Query Intro
	if intro != "" {
		var keywords []string
		keywordsMatrix[1], keywords = getKeywords(intro)
		selects = append(selects, unionByWords(keywords, "sum(count)", func(word string) string {
			return "select id, count as count from datapacks_intro_ii where word like '" + word + "%'"
		}, "as _intro group by id having count(*) >= "+strconv.Itoa(len(keywords))))
	}
	// Query Author
	if author != "" {
		keywordsMatrix[2] = &author
		sql = sql.Where("authors.author_name like '" + author + "%'")
	}
	sql, table := datapackFilter(sql, source, version, postTimeRange, updateTimeRange) // Filter, Using Joined Table
	table = "(select id, sum(count) as count from (" + strings.Join(selects, " union all ") + ") as results group by id having count(*) >= " + strconv.Itoa(len(selects)) + " ) as ids left join " + table + " on ids.id = datapacks.id"
	fmt.Println(table)
	sql.Table(table).Count(&total).Order("count DESC").Offset(offset).Limit(limit).Find(&datapacks) // Find All
	// Count and Mark Keywords
	for i := 0; i < len(datapacks); i++ {
		datapacks[i].Initialize()
		if name != "" {
			KeyWordHighlight(&datapacks[i].Name, keywordsMatrix[0])
		}
		if intro != "" {
			KeyWordHighlight(&datapacks[i].Intro, keywordsMatrix[1])
		}
		if author != "" {
			KeyWordHighlight(&datapacks[i].Author.AuthorName, keywordsMatrix[2])
		}
	}
	return &datapacks, total
}

// Rand
func GetRandID(table string) string {
	var id []string
	err := db.Raw("select id from "+table+" order by rand() limit 1;").Pluck("id", &id).Error
	if err != nil {
		panic(err)
	}
	if len(id) == 0 {
		panic("result empty.")
	}
	return id[0]
}
func GetRandDatapack(language string) *Datapack {
	return GetDatapack(language, GetRandID("datapacks"))
}
func GetRandAuthor(language string) *Author {
	return GetAuthor(language, GetRandID("authors"))
}
func GetRandTag(page int, language string) (*Tag, int) {
	tag, total := GetTag(page, language, GetRandID("tags"))
	return tag, total
}

// Thumb
func Thumb(table string, id string) {
	err := db.Table(table).Where("id = ?", id).UpdateColumn("thumb", gorm.Expr("thumb + 1")).Error
	if err != nil {
		fmt.Println(err)
	}
}
