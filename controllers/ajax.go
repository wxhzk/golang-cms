package controllers

import (
	"encoding/json"
	"image"
	"io"
	"mime/multipart"

	"github.com/dionyself/golang-cms/models"
	"github.com/dionyself/golang-cms/utils"
)

type AjaxController struct {
	BaseController
}

// generic Ajax
func (CTRL *AjaxController) PostImage() {
	form := new(models.ImageForm)
	if err := CTRL.ParseForm(form); err != nil {
		CTRL.Abort("401")
	} else {
		if form.Validate() {
			if rawFile, fileHeader, err := CTRL.GetFile("File"); err == nil && utils.Contains(utils.SuportedMimeTypes["images"], fileHeader.Header["Conten-Type"][0]) {
				defer rawFile.Close()
				newSession := utils.GetRandomString(16)
				go CTRL.uploadAndRegisterIMG(newSession, rawFile, fileHeader, form)
				response := map[string]string{"status": "started", "sessionId": newSession}
				CTRL.Data["json"] = &response
			}
		}
	}
	CTRL.ServeJSON()
}

func (CTRL *AjaxController) uploadAndRegisterIMG(sessionKey string, img io.Reader, fileHeader *multipart.FileHeader, form *models.ImageForm) {
	var croppedImg image.Image
	var targets map[string][2]int
	status := map[string]string{}
	json.Unmarshal([]byte(form.Targets), &targets)
	CTRL.cache.Set(sessionKey, "value", 30)
	for target, coords := range targets {
		if !utils.ContainsKey(utils.ImageSizes, target) {
			break
		}
		croppedImg, _ = utils.CropImage(img, fileHeader.Header["Conten-Type"][0], target, coords)
		if croppedImg != nil {
			utils.UploadImage(target, croppedImg)
		}
		status[target] = "done"
		value, _ := json.Marshal(status)
		CTRL.cache.Set(sessionKey, string(value[:]), 30)
	}
	user := CTRL.Data["user"].(models.User)
	newImg := new(models.Image)
	newImg.User = &user
	CTRL.db.Insert(newImg)
	CTRL.db.Insert(user)
	status["image_id"] = string(user.Id)
	value, _ := json.Marshal(status)
	CTRL.cache.Set(sessionKey, string(value[:]), 30)
}

func (CTRL *AjaxController) GetImageUploadStatus() {
	data := map[string]string{}
	if err := json.Unmarshal(CTRL.Ctx.Input.RequestBody, &data); err != nil {
		CTRL.Ctx.Output.SetStatus(400)
	}
	if status, err := CTRL.cache.GetString("sessionKey", 30); err == false {
		json.Unmarshal([]byte(status), &data)
	}
	CTRL.Data["json"] = &data
	CTRL.ServeJSON()
}
