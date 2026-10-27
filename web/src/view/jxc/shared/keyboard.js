/** 判断事件目标是否是一个正在输入的可编辑元素。键盘快捷键不应劫持用户打字。 */
export const isTypingTarget = (target) => {
  if (!target || !(target instanceof HTMLElement)) return false
  const tag = target.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable || target.closest('[contenteditable="true"]') != null
}
