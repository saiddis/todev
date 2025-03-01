document.querySelectorAll(".time").forEach(function(time) {
	ms = Date.now() - Date.parse(time.innerHTML)
	let humanizedTime = "Now"
	if (ms >= 86400000) {
		humanizedTime = `${Math.floor(ms / 86400000)} days ago`
	} else if (Math.floor(ms >= 3600000)) {
		humanizedTime = `${Math.floor(ms / 3600000)} hours ago`
	} else if (ms >= 60000) {
		humanizedTime = `${Math.floor(ms / 60000)} minutes ago`
	} else if (ms >= 5000) {
		humanizedTime = `${Math.floor(ms / 1000)} seconds ago`
	}
	time.innerHTML = humanizedTime
})

const createRepoButton = document.getElementById('create-repo-button')
const createRepoForm = document.getElementById('create-repo-form')
const createRepoInput = createRepoForm.querySelector('.form__input')
const submitButton = createRepoForm.querySelector('button')
const crossIcon = createRepoForm.querySelector('.icon-cross')
createRepoButton.onclick = function() {
	createRepoForm.style.display = 'flex'
	createRepoForm.style.left = document.documentElement.clientWidth / 2 - createRepoForm.clientWidth / 2 + 'px'
	createRepoForm.style.top = document.documentElement.clientHeight / 2 - createRepoForm.clientHeight / 2 + 'px'
	createRepoInput.focus()
}

submitButton.onclick = () => {
	if (createRepoInput.value.length < 4) {
		return false
	}
}

crossIcon.onclick = () => {
	createRepoForm.style.display = 'none'
	createRepoInput.value = ''
}
