export function setTitle(title : undefined | string) {
	document.title = (typeof title === 'undefined' || title === '')
		? 'Thirty Six Toyota'
		: `${title} | Thirty Six Toyota`
}
