// Backpack shapes travel as a string of '1' (open) and '0' (closed) cells, row by
// row. An empty string is the plain rectangular grid where every cell is open.

export function fullMask(width, height) {
  return '1'.repeat(Math.max(0, width * height))
}

// The mask to draw on: a shaped backpack as is, a plain grid as all-open cells.
export function expandMask(mask, width, height) {
  return mask && mask.length === width * height ? mask : fullMask(width, height)
}

export function isCellOpen(mask, width, x, y) {
  if (!mask) return true
  return mask[y * width + x] === '1'
}

export function countOpen(mask) {
  let open = 0
  for (const cell of mask) if (cell === '1') open += 1
  return open
}

// Keeps the shape anchored top-left; cells added by growing the grid start as `fill`.
export function resizeMask(mask, oldWidth, oldHeight, newWidth, newHeight, fill = '0') {
  let next = ''
  for (let y = 0; y < newHeight; y += 1) {
    for (let x = 0; x < newWidth; x += 1) {
      next += x < oldWidth && y < oldHeight ? (mask[y * oldWidth + x] ?? fill) : fill
    }
  }
  return next
}

// "x,y" keys of the cells under items sitting on the grid, so an editor can keep them open.
export function occupiedCells(character) {
  const equipped = new Set((character?.equipment ?? []).map(slot => slot.inventory_item_id))
  const cells = new Set()
  for (const entry of character?.inventory ?? []) {
    if (equipped.has(entry.id)) continue
    let itemWidth = Math.max(1, entry.item?.grid_width || 1)
    let itemHeight = Math.max(1, entry.item?.grid_height || 1)
    if (entry.is_rotated) [itemWidth, itemHeight] = [itemHeight, itemWidth]
    for (let y = entry.grid_y; y < entry.grid_y + itemHeight; y += 1) {
      for (let x = entry.grid_x; x < entry.grid_x + itemWidth; x += 1) {
        cells.add(`${x},${y}`)
      }
    }
  }
  return cells
}
