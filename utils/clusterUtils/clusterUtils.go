package clusterUtils

import (
	"bytes"
	"dst-admin-go/config/database"
	"dst-admin-go/model"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func GetCluster(clusterName string) *model.Cluster {
	db := database.DB
	cluster := &model.Cluster{}
	db.Where("cluster_name=?", clusterName).First(cluster)
	return cluster
}

func GetClusterFromGin(ctx *gin.Context) *model.Cluster {
	clusterName := ctx.GetHeader("Cluster")
	log.Print(ctx.Request.RequestURI, "cluster: ", clusterName)
	db := database.DB
	cluster := &model.Cluster{}
	db.Where("cluster_name=?", clusterName).First(cluster)
	return cluster
}

func GetDstServerInfo(clusterName string) []DstHomeInfo {

	d := "{\"page\": 1,\"paginate\": 10,\"sort_type\": \"name\",\"sort_way\": 1,\"search_type\": 1,\"search_content\": \"%s\",\"mod\": 1}"
	d2 := fmt.Sprintf(d, clusterName)
	data := []byte(d2)
	// 创建HTTP请求
	url := "https://dst.liuyh.com/index/serverlist/getserverlist.html"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println("33333", err)
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json")
	// 发送HTTP请求
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Println("2222", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// 处理错误
		log.Println("1111", err)
	}
	s := string(body)
	s = s[1 : len(s)-1]
	s = strings.Replace(s, "\\", "", -1)
	fmt.Println(s)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(s), &result)
	if err != nil {
		fmt.Println(err)
	}
	if !result["success"].(bool) {
		return []DstHomeInfo{}
	}
	homeData := result["successinfo"].(map[string]interface{})["data"].([]interface{})
	if len(homeData) == 0 {
		return []DstHomeInfo{}
	}
	var homeDataList []DstHomeInfo
	for _, d := range homeData {
		row := d.([]interface{})[0].(string)
		connected := d.([]interface{})[5].(float64)
		maxConnect := d.([]interface{})[6].(float64)
		mode := d.([]interface{})[8].(string)
		mods := d.([]interface{})[9].(float64)
		name := d.([]interface{})[10].(string)
		password := d.([]interface{})[11].(float64)
		season := d.([]interface{})[14].(string)
		h := DstHomeInfo{
			Row:        row,
			Connected:  connected,
			MaxConnect: maxConnect,
			Mode:       mode,
			Mods:       mods,
			Name:       name,
			Password:   password,
			Season:     season,
		}
		homeDataList = append(homeDataList, h)
	}
	return homeDataList
}

type DstHomeInfo struct {
	Row        string
	Connected  float64
	MaxConnect float64
	Mode       string
	Mods       float64
	Name       string
	Password   float64
	Season     string
}