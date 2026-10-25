import {
  AppWindow,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  CircleDot,
  House,
  Keyboard,
  ListVideo,
  Moon,
  Play,
  Power,
  Search,
  Settings,
  Undo2,
  Volume1,
  Volume2,
  VolumeX,
} from 'lucide-react'
import type { IconName } from '../lib/commands'

const MAP: Record<IconName, typeof Power> = {
  app: AppWindow,
  volume: Volume2,
  'volume-up': Volume2,
  'volume-down': Volume1,
  mute: VolumeX,
  power: Power,
  home: House,
  back: Undo2,
  ok: CircleDot,
  up: ChevronUp,
  down: ChevronDown,
  left: ChevronLeft,
  right: ChevronRight,
  play: Play,
  settings: Settings,
  channel: ListVideo,
  text: Keyboard,
  search: Search,
  moon: Moon,
}

export function CommandIcon({ name, size = 16 }: { name: IconName; size?: number }) {
  const Icon = MAP[name]
  return <Icon size={size} strokeWidth={1.8} />
}
