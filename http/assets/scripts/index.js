document.querySelectorAll(".time").forEach(function(time) {
	ms = Date.now() - Date.parse(time.innerHTML)
	let humanizedTime = "Now"
	if (ms >= 86400000) {
		humanizedTime = `${Math.floor(ms / 86400000)} days ago`
	} else if (Math.floor(ms >= 3600000)) {
		humanizedTime = `${Math.floor(ms / 3600000)} hours ago`
	} else if (ms >= 60000) {
		humanizedTime = `${Math.floor(ms / 60000)} minutes ago`
	} else if (m >= 5000) {
		humanizedTime = `${Math.floor(ms / 1000)} seconds ago`
	}
	time.innerHTML = humanizedTime
})
