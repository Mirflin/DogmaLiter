import { push } from 'notivue'

function toPayload(value, fallbackTitle) {
  if (typeof value === 'string') {
    return {
      title: fallbackTitle,
      message: value,
    }
  }

  return {
    title: fallbackTitle,
    ...value,
  }
}

export function getErrorMessage(error, fallback = 'Something went wrong') {
  return error?.response?.data?.error || error?.message || fallback
}

export const notify = {
  success(value) {
    return push.success(toPayload(value, 'Success'))
  },
  error(value, fallback) {
    const payload = typeof value === 'string'
      ? value
      : getErrorMessage(value, fallback)

    return push.error(toPayload(payload, 'Action failed'))
  },
  info(value) {
    return push.info(toPayload(value, 'Heads up'))
  },
  warning(value) {
    return push.warning(toPayload(value, 'Warning'))
  },
}