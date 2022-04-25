package main

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

/* LinkedIn App Implementation */

//Structs declaration

type PostCreated struct {
	Id string `json:"id"`
}

type OauthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

type UserDetails struct {
	UserName string `json:"localizedFirstName"`
	UserId   string `json:"id"`
}

type MediaUploadHttpRequest struct {
	UploadURL string `json:"uploadUrl"`
}
type UploadMechanism struct {
	UploadHttpRequest MediaUploadHttpRequest `json:"com.linkedin.digitalmedia.uploading.MediaUploadHttpRequest"`
}
type DigitalAsset struct {
	Uploadmechanism UploadMechanism `json:"uploadMechanism"`
	AssetID         string          `json:"asset"`
}
type DigitalAssetValue struct {
	Values DigitalAsset `json:"value"`
}

//Global variable declaration
var userId string
var global_access_token string

//Main method declaration
func main() {
	var (
		buf    bytes.Buffer
		logger = log.New(&buf, "Logger : ", log.Lshortfile)
	)
	fmt.Println("Application Started")

	applicationPort := "9000"
	fmt.Println("Application Parameters " + applicationPort)

	renderer := gin.Default()
	renderer.LoadHTMLGlob("templates/*.tmpl")
	fmt.Println("Loaded Templates")

	//GIN Handlers declaration
	renderer.GET("/", func(ctx *gin.Context) {
		cookie, err := ctx.Cookie("access_token")
		//This condition should reverse
		if err == nil && cookie == "" {
			ctx.HTML(http.StatusOK, "index.tmpl", gin.H{})
		} else {
			access_token := cookie
			client := &http.Client{}
			requestObject, err := http.NewRequest("GET", "https://api.linkedin.com/v2/me", nil)
			if err != nil {
				logger.Fatal(err)
				return
			}
			query := requestObject.URL.Query()
			query.Add("oauth2_access_token", access_token)
			requestObject.URL.RawQuery = query.Encode()
			userData, errorObject := client.Do(requestObject)
			if errorObject != nil {
				log.Fatal(err)
			}
			defer userData.Body.Close()
			body, err := ioutil.ReadAll(userData.Body)
			if err != nil {
				logger.Fatal(err)
				return
			}
			var userDetail UserDetails
			json.Unmarshal([]byte(body), &userDetail)
			global_access_token = access_token
			userId = userDetail.UserId
			ctx.HTML(http.StatusOK, "uploadPost.tmpl", gin.H{"userId": userDetail.UserId, "name": userDetail.UserName})
		}
	})

	renderer.GET("/callback/linkedin", func(ctx *gin.Context) {
		fmt.Println("Callback received - Auth Code" + ctx.Query("code"))
		authCode := ctx.Query("code")
		apiEndpoint := "https://www.linkedin.com/oauth/v2/accessToken"
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", authCode)
		data.Set("client_id", "774yimr2zvlnep")
		data.Set("redirect_uri", "https://linkedin-go-app.herokuapp.com/callback/linkedin")
		client := &http.Client{}
		requestObject, err := http.NewRequest("POST", apiEndpoint, strings.NewReader(data.Encode()))
		if err != nil {
			log.Fatal(err)
		}
		authHeader := b64.StdEncoding.EncodeToString([]byte("774yimr2zvlnep:JYZS2uxXnj8nxuh6"))
		requestObject.Header.Add("Authorization", "Basic "+authHeader)
		requestObject.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		requestObject.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
		response, err := client.Do(requestObject)
		if err != nil {
			log.Fatal(err)
		}
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		var oauthResponse OauthResponse
		json.Unmarshal(body, &oauthResponse)
		ctx.SetCookie("access_token", oauthResponse.AccessToken, 0, "/", "https://linkedin-go-app.herokuapp.com", false, false)
		if err != nil {
			log.Fatal(err)
		}
		ctx.Redirect(http.StatusFound, "/")
	})

	//This should be uploadPost
	renderer.POST("/uploadPost", func(ctx *gin.Context) {
		fmt.Println("Inside upload post method")

		/*filePlain, err := ctx.FormFile("myfile")
		if err != nil {
			ctx.String(http.StatusBadRequest, fmt.Sprintf("File Error : %s", err.Error()))
			return
		}
		extension := filepath.Ext(filePlain.Filename)
		newFileName := uuid.New().String() + extension
		fileRelativePath := "images/" + newFileName
		if err := ctx.SaveUploadedFile(filePlain, fileRelativePath); err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "Unable to save the uploaded file",
			})
			return
		}*/

		/*uploadUrl, assetId := registerImageUplaod()
		statusCode := UploadImageToLinkedIn(fileRelativePath, uploadUrl)
		if statusCode == http.StatusCreated {
			ctx.String(http.StatusOK, string("The LinkedIn post image has been updated ."))
			return
		}*/
		postContent := ctx.PostForm("postContent")

		postId := createPost(postContent)
		ctx.HTML(http.StatusOK, "postcreated.tmpl", gin.H{"id": postId})
	})

	renderer.Run(":" + applicationPort)
}

func createPost(postContent string) string {
	dataString := `{
		"author": "urn:li:person:<user_id>",
		"lifecycleState": "PUBLISHED",
		"specificContent": {
			"com.linkedin.ugc.ShareContent": {
				"shareCommentary": {
					"attributes": [],
					"text": "<post_content>"
				},
				"shareMediaCategory": "NONE"
			}
		},
		"targetAudience": {
			"targetedEntities": [
				{
					"geoLocations": [
						"urn:li:geo:103644278"
					],
					"seniorities": [
						"urn:li:seniority:3"
					]
				}
			]
		},
		"visibility": {
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"
		}
	}`
	endPoint := "https://api.linkedin.com/v2/ugcPosts?oauth2_access_token=" + global_access_token
	latestData := strings.Replace(dataString, "<user_id>", userId, -1)
	newData := strings.Replace(latestData, "<post_content>", postContent, -1)
	client := &http.Client{}
	requestObject, err := http.NewRequest("POST", endPoint, bytes.NewBuffer([]byte(newData)))
	if err != nil {
		log.Fatal(err)
	}
	response, error := client.Do(requestObject)
	if error != nil {
		log.Fatal(error)
	}
	fmt.Println("Response status code " + strconv.Itoa(response.StatusCode))
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	fmt.Println(string(body))
	if response.StatusCode == http.StatusCreated {
		fmt.Println("Post has been created")
	}
	var postDetails PostCreated
	json.Unmarshal([]byte(body), &postDetails)
	return postDetails.Id
}

/*func registerImageUplaod() (UploadUrl string, AssetID string) {
	dataString := `{
		"registerUploadRequest":{
		   "owner":"urn:li:person:<user_id>",
		   "recipes":[
			  "urn:li:digitalmediaRecipe:feedshare-image"
		   ],
		   "serviceRelationships":[
			  {
				 "identifier":"urn:li:userGeneratedContent",
				 "relationshipType":"OWNER"
			  }
		   ],
		   "supportedUploadMechanism":[
			  "SYNCHRONOUS_UPLOAD"
		   ]
		}
	 }`
	endPoint := "https://api.linkedin.com/v2/assets?action=registerUpload&oauth2_access_token=" + global_access_token + ""
	newData := strings.Replace(dataString, "<user_id>", userId, -1)
	client := &http.Client{}
	requestObject, err := http.NewRequest("POST", endPoint, bytes.NewBuffer([]byte(newData)))
	if err != nil {
		log.Fatal(err)
	}
	response, error := client.Do(requestObject)
	if error != nil {
		log.Fatal(error)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	var DigitalAssetValues DigitalAssetValue
	json.Unmarshal(body, &DigitalAssetValues)
	return DigitalAssetValues.Values.Uploadmechanism.UploadHttpRequest.UploadURL, DigitalAssetValues.Values.AssetID
}*/

/*func UploadImageToLinkedIn(filePath string, uploadUrl string) int {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(part, file)
	writer.Close()

	request, requestError := http.NewRequest("PUT", uploadUrl+"?oauth2_access_token="+global_access_token, body)
	if requestError != nil {
		log.Fatal(requestError)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	response, responseError := client.Do(request)
	if responseError != nil {
		log.Fatal(responseError)
	}
	body2 := &bytes.Buffer{}
	_, errNew := body2.ReadFrom(response.Body)
	if errNew != nil {
		log.Fatal(errNew)
	}
	response.Body.Close()
	fmt.Println(response.StatusCode)
	fmt.Println(response.Header)
	fmt.Println(body)
	return response.StatusCode
}*/

/*func UploadImageToLinkedIn(filePath string, UploadUrl string) int {
	client := &http.Client{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	upload, uplaoderror := writer.CreateFormFile("Photo", filePath)
	if uplaoderror != nil {
		log.Fatal(uplaoderror)
	}
	file, fileerror := os.Open(filePath)
	if fileerror != nil {
		log.Fatal(fileerror)
	}
	_, copyerror := io.Copy(upload, file)
	if copyerror != nil {
		log.Fatal(copyerror)
	}
	request, requesterror := http.NewRequest("POST", UploadUrl, bytes.NewReader(body.Bytes()))
	if requesterror != nil {
		log.Fatal(copyerror)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Add("oauth2_access_token", global_access_token)
	response, _ := client.Do(request)

	if response.StatusCode != http.StatusCreated {
		log.Fatal("Image not uploaded " + strconv.Itoa(response.StatusCode))
	}
	return response.StatusCode
}*/

/* LinkedIn App Implementation - ends*/
