const nav = document.querySelector("nav")
const main = document.querySelector("main")
main.style.top = nav.clientHeight + "px"
document.body.style.height = document.documentElement.clientHeight + nav.clientHeight + "px"

document.addEventListener("DOMContentLoaded", () => {
	document.body.addEventListener("htmx:beforeSwap", function(event) {
		if (event.detail.xhr.status === 422) {
			event.detail.shouldSwap = true;
			event.detail.isError = false;
		}
	})
})

const cookies = document.cookie.split("; ").reduce((acc, cookie) => {
	const [name, value] = cookie.split("=")
	acc[name] = decodeURIComponent(value);
	return acc;
}, {})

const avatarURL = cookies["avatar"]

if (avatarURL) {
	const avatarImg = document.querySelector(".avatar")
	avatarImg.src = avatarURL

}
