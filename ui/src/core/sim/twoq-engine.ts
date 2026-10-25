import type { CacheEntry } from '../types';

interface TwoQNode {
  key: string;
  value: string;
  qType: 'A1' | 'Am';
  expiresAt: number | null;
  createdAt: number;
  prev: TwoQNode | null;
  next: TwoQNode | null;
}

class TwoQList {
  public head: TwoQNode;
  public tail: TwoQNode;
  public len: number;

  constructor() {
    this.head = {
      key: '',
      value: '',
      qType: 'A1',
      expiresAt: null,
      createdAt: 0,
      prev: null,
      next: null,
    };
    this.tail = {
      key: '',
      value: '',
      qType: 'A1',
      expiresAt: null,
      createdAt: 0,
      prev: null,
      next: null,
    };
    this.head.next = this.tail;
    this.tail.prev = this.head;
    this.len = 0;
  }

  public pushFront(node: TwoQNode): void {
    node.prev = this.head;
    node.next = this.head.next;
    if (this.head.next) {
      this.head.next.prev = node;
    }
    this.head.next = node;
    this.len++;
  }

  public remove(node: TwoQNode): void {
    if (node.prev) {
      node.prev.next = node.next;
    }
    if (node.next) {
      node.next.prev = node.prev;
    }
    node.prev = null;
    node.next = null;
    this.len--;
  }

  public moveToFront(node: TwoQNode): void {
    this.remove(node);
    this.pushFront(node);
  }

  public popTail(): TwoQNode | null {
    if (this.len === 0 || this.tail.prev === this.head || !this.tail.prev) {
      return null;
    }
    const oldest = this.tail.prev;
    this.remove(oldest);
    return oldest;
  }
}

/**
 * 2Q (Two-Queue / Scan-Resistant) cache engine.
 * Direct TypeScript port of `internal/cache/twoq.go`.
 * A1: FIFO queue for probation/new entries (default 25% of capacity).
 * Am: LRU queue for promoted/frequent entries (default 75% of capacity).
 */
export class TwoQEngine {
  private capacity: number;
  private a1Cap: number;
  private amCap: number;
  private items: Map<string, TwoQNode> = new Map();
  private a1: TwoQList;
  private am: TwoQList;

  constructor(capacity: number, a1Ratio: number = 0.25) {
    if (capacity <= 0) {
      throw new Error('Capacity must be greater than 0');
    }
    this.capacity = capacity;

    if (capacity <= 1) {
      this.a1Cap = 1;
      this.amCap = 1;
    } else {
      this.a1Cap = Math.max(1, Math.floor(capacity * a1Ratio));
      this.amCap = Math.max(1, capacity - this.a1Cap);
    }

    this.a1 = new TwoQList();
    this.am = new TwoQList();
  }

  public getCapacity(): number {
    return this.capacity;
  }

  public getA1Capacity(): number {
    return this.a1Cap;
  }

  public getAmCapacity(): number {
    return this.amCap;
  }

  public size(): number {
    return this.items.size;
  }

  public a1Size(): number {
    return this.a1.len;
  }

  public amSize(): number {
    return this.am.len;
  }

  private isExpired(node: TwoQNode, now: number = Date.now()): boolean {
    return node.expiresAt !== null && now >= node.expiresAt;
  }

  private promoteToAm(node: TwoQNode): void {
    this.a1.remove(node);

    // If Am is full or overall capacity is exceeded, evict from Am
    if (this.am.len >= this.amCap || this.items.size > this.capacity) {
      this.evictAm();
    }

    node.qType = 'Am';
    this.am.pushFront(node);
  }

  private evictA1(): TwoQNode | null {
    const oldest = this.a1.popTail();
    if (oldest) {
      this.items.delete(oldest.key);
    }
    return oldest;
  }

  private evictAm(): TwoQNode | null {
    const lru = this.am.popTail();
    if (lru) {
      this.items.delete(lru.key);
    }
    return lru;
  }

  private removeNode(node: TwoQNode): void {
    this.items.delete(node.key);
    if (node.qType === 'A1') {
      this.a1.remove(node);
    } else {
      this.am.remove(node);
    }
  }

  public set(
    key: string,
    value: string,
    ttlMs?: number | null
  ): { evicted: boolean; evictedKey?: string; promoted?: boolean } {
    const expiresAt = ttlMs && ttlMs > 0 ? Date.now() + ttlMs : null;

    const existing = this.items.get(key);
    if (existing) {
      existing.value = value;
      existing.expiresAt = expiresAt;
      if (existing.qType === 'A1') {
        // Updating existing A1 key counts as access -> promote to Am
        this.promoteToAm(existing);
        return { evicted: false, promoted: true };
      } else {
        this.am.moveToFront(existing);
        return { evicted: false };
      }
    }

    let evicted = false;
    let evictedKey: string | undefined;

    // New key: insert into A1. If A1 is full, evict oldest from A1
    if (this.a1.len >= this.a1Cap) {
      const removed = this.evictA1();
      if (removed) {
        evicted = true;
        evictedKey = removed.key;
      }
    }

    // If overall cache capacity is reached, evict to maintain bounds
    if (this.items.size >= this.capacity) {
      if (this.a1.len > 0) {
        const removed = this.evictA1();
        if (removed) {
          evicted = true;
          evictedKey = removed.key;
        }
      } else if (this.am.len > 0) {
        const removed = this.evictAm();
        if (removed) {
          evicted = true;
          evictedKey = removed.key;
        }
      }
    }

    const newNode: TwoQNode = {
      key,
      value,
      qType: 'A1',
      expiresAt,
      createdAt: Date.now(),
      prev: null,
      next: null,
    };

    this.items.set(key, newNode);
    this.a1.pushFront(newNode);

    return { evicted, evictedKey };
  }

  public get(
    key: string
  ): { value: string | null; found: boolean; expired: boolean; promoted?: boolean; qType?: 'A1' | 'Am' } {
    const node = this.items.get(key);
    if (!node) {
      return { value: null, found: false, expired: false };
    }

    if (this.isExpired(node)) {
      this.removeNode(node);
      return { value: null, found: false, expired: true };
    }

    let promoted = false;
    if (node.qType === 'A1') {
      // Second access: promote from A1 to Am!
      this.promoteToAm(node);
      promoted = true;
    } else {
      // Already in Am: move to MRU
      this.am.moveToFront(node);
    }

    return {
      value: node.value,
      found: true,
      expired: false,
      promoted,
      qType: node.qType,
    };
  }

  public delete(key: string): { deleted: boolean; expired: boolean } {
    const node = this.items.get(key);
    if (!node) {
      return { deleted: false, expired: false };
    }

    const expired = this.isExpired(node);
    this.removeNode(node);

    return { deleted: !expired, expired };
  }

  public clear(): void {
    this.items.clear();
    this.a1 = new TwoQList();
    this.am = new TwoQList();
  }

  /**
   * Introspection snapshots for 2Q visualizer.
   */
  public getA1Entries(): CacheEntry[] {
    const result: CacheEntry[] = [];
    const now = Date.now();
    let curr = this.a1.head.next;
    while (curr && curr !== this.a1.tail) {
      if (!this.isExpired(curr, now)) {
        result.push({
          key: curr.key,
          value: curr.value,
          qType: 'A1',
          expiresAt: curr.expiresAt,
          createdAt: curr.createdAt,
        });
      }
      curr = curr.next;
    }
    return result;
  }

  public getAmEntries(): CacheEntry[] {
    const result: CacheEntry[] = [];
    const now = Date.now();
    let curr = this.am.head.next;
    while (curr && curr !== this.am.tail) {
      if (!this.isExpired(curr, now)) {
        result.push({
          key: curr.key,
          value: curr.value,
          qType: 'Am',
          expiresAt: curr.expiresAt,
          createdAt: curr.createdAt,
        });
      }
      curr = curr.next;
    }
    return result;
  }
}
