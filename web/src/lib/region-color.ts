export const HERALDIC_COLORS = [
  '#1d4ed8',
  '#b91c1c',
  '#15803d',
  '#d97706',
  '#7c3aed',
  '#374151',
] as const

export const REGION_PATTERNS = [
  'diagonal',
  'vertical',
  'horizontal',
  'cross',
  'dots',
  'diamonds',
] as const

export type RegionPattern = (typeof REGION_PATTERNS)[number]

export const REGION_PATTERN_BACKGROUNDS: Record<RegionPattern, string> = {
  diagonal:
    'repeating-linear-gradient(45deg, transparent 0 4px, rgba(255,255,255,0.65) 4px 5px)',
  vertical:
    'repeating-linear-gradient(90deg, transparent 0 4px, rgba(255,255,255,0.65) 4px 5px)',
  horizontal:
    'repeating-linear-gradient(0deg, transparent 0 4px, rgba(255,255,255,0.65) 4px 5px)',
  cross:
    'repeating-linear-gradient(0deg, transparent 0 5px, rgba(255,255,255,0.55) 5px 6px), repeating-linear-gradient(90deg, transparent 0 5px, rgba(255,255,255,0.55) 5px 6px)',
  dots: 'radial-gradient(circle, rgba(255,255,255,0.8) 1px, transparent 1.5px) 0 0 / 6px 6px',
  diamonds:
    'repeating-linear-gradient(45deg, transparent 0 6px, rgba(255,255,255,0.65) 6px 7px, transparent 7px 12px), repeating-linear-gradient(-45deg, transparent 0 6px, rgba(255,255,255,0.65) 6px 7px, transparent 7px 12px)',
}

export interface RegionStyle {
  fill: string
  pattern: RegionPattern | null
}

export function regionStyle(index: number, total: number): RegionStyle {
  const fill = HERALDIC_COLORS[index % HERALDIC_COLORS.length]
  if (index < HERALDIC_COLORS.length || total <= HERALDIC_COLORS.length) {
    return { fill, pattern: null }
  }

  const patternIndex = Math.floor(index / HERALDIC_COLORS.length) - 1
  return {
    fill,
    pattern: REGION_PATTERNS[patternIndex % REGION_PATTERNS.length],
  }
}
