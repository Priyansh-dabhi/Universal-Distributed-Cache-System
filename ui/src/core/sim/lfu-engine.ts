import type { CacheEntry } from '../types';

interface LFUNode {
  key: string;
  value: string;
  freq: number;
  expiresAt: number | null;
  createdAt: number;
  prev: LFUNode | null;
  next: LFUNode | null;
}

class LFUList {
  public head: LFUNode;
  public tail: LFUNode;
  public len: number;

  constructor() {
    this.head = {
      key: '',
      value: '',
      freq: 0,
      expiresAt: null,
      createdAt: 0,
      prev: null,
      next: null,
    };
    this.tail = {
      key: '',
      value: '',
      freq: 0,
      expiresAt: null,
      createdAt: 0,
      prev: null,
      next: null,
    };
    this.head.next = this.tail;
    this.tail.prev = this.head;
    this.len = 0;
  }

  public pushFront(node: LFUNode): void {
    node.prev = this.head;
    node.next = this.head.next;
    if (this.head.next) {
      this.head.next.prev = node;
    }
    this.head.next = node;
    this.len++;
  }

  public remove(node: LFUNode): void {
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

  public popTail(): LFUNode | null {
    if (this.len === 0 || this.tail.prev === this.head || !this.tail.prev) {
      return null;
    }
    const lru = this.tail.prev;
    this.remove(lru);
    return lru;
  }

  public isEmpty(): boolean {
    return this.len === 0;
  }
}

/**
 * LFU (Least Frequently Used) cache engine with LRU tie-breaking.
 * Direct TypeScript port of `internal/cache/lfu.go`.
 */
export class LFUEngine {
  private capacity: number;
  private items: Map<string, LFUNode> = new Map();
  private freqBuckets: Map<number, LFUList> = new Map();
  private minFreq: number = 0;

  constructor(capacity: number) {
    if (capacity <= 0) {
      throw new Error('Capacity must be greater than 0');
    }
    this.capacity = capacity;
  }

  public getCapacity(): number {
    return this.capacity;
  }

  public size(): number {
    return this.items.size;
  }

  private isExpired(node: LFUNode, now: number = Date.now()): boolean {
    return node.expiresAt !== null && now >= node.expiresAt;
  }

  private incrementFreq(node: LFUNode): void {
    const oldFreq = node.freq;
    const oldBucket = this.freqBuckets.get(oldFreq);
    if (oldBucket) {
      oldBucket.remove(node);
      if (oldBucket.isEmpty()) {
        this.freqBuckets.delete(oldFreq);
        if (this.minFreq === oldFreq) {
          this.minFreq = oldFreq + 1;
        }
      }
    }

    node.freq++;
    const newFreq = node.freq;
    let newBucket = this.freqBuckets.get(newFreq);
    if (!newBucket) {
      newBucket = new LFUList();
      this.freqBuckets.set(newFreq, newBucket);
    }
    newBucket.pushFront(node);
  }

  private evict(): LFUNode | null {
    const bucket = this.freqBuckets.get(this.minFreq);
    if (!bucket || bucket.isEmpty()) {
      return null;
    }

    const evicted = bucket.popTail();
    if (evicted) {
      this.items.delete(evicted.key);
    }

    if (bucket.isEmpty()) {
      this.freqBuckets.delete(this.minFreq);
    }

    return evicted;
  }

  private removeNode(node: LFUNode): void {
    this.items.delete(node.key);
    const bucket = this.freqBuckets.get(node.freq);
    if (bucket) {
      bucket.remove(node);
      if (bucket.isEmpty()) {
        this.freqBuckets.delete(node.freq);
        if (this.minFreq === node.freq) {
          if (this.items.size === 0) {
            this.minFreq = 0;
          } else {
            let min = 0;
            for (const f of this.freqBuckets.keys()) {
              if (min === 0 || f < min) {
                min = f;
              }
            }
            this.minFreq = min;
          }
        }
      }
    }
  }

  public set(
    key: string,
    value: string,
    ttlMs?: number | null
  ): { evicted: boolean; evictedKey?: string } {
    const expiresAt = ttlMs && ttlMs > 0 ? Date.now() + ttlMs : null;

    const existing = this.items.get(key);
    if (existing) {
      existing.value = value;
      existing.expiresAt = expiresAt;
      this.incrementFreq(existing);
      return { evicted: false };
    }

    let evicted = false;
    let evictedKey: string | undefined;

    if (this.items.size >= this.capacity) {
      const removed = this.evict();
      if (removed) {
        evicted = true;
        evictedKey = removed.key;
      }
    }

    const newNode: LFUNode = {
      key,
      value,
      freq: 1,
      expiresAt,
      createdAt: Date.now(),
      prev: null,
      next: null,
    };

    this.items.set(key, newNode);
    let b1 = this.freqBuckets.get(1);
    if (!b1) {
      b1 = new LFUList();
      this.freqBuckets.set(1, b1);
    }
    b1.pushFront(newNode);
    this.minFreq = 1;

    return { evicted, evictedKey };
  }

  public get(key: string): { value: string | null; found: boolean; expired: boolean; freq?: number } {
    const node = this.items.get(key);
    if (!node) {
      return { value: null, found: false, expired: false };
    }

    if (this.isExpired(node)) {
      this.removeNode(node);
      return { value: null, found: false, expired: true };
    }

    this.incrementFreq(node);
    return { value: node.value, found: true, expired: false, freq: node.freq };
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

  public getFrequency(key: string): number | null {
    const node = this.items.get(key);
    if (!node || this.isExpired(node)) {
      return null;
    }
    return node.freq;
  }

  public clear(): void {
    this.items.clear();
    this.freqBuckets.clear();
    this.minFreq = 0;
  }

  /**
   * Snapshot of frequency buckets for LFU visualizer.
   * Sorted by frequency ascending: [{ freq: 1, entries: [...] }, { freq: 2, ... }]
   */
  public getBucketSnapshots(): { freq: number; entries: CacheEntry[] }[] {
    const result: { freq: number; entries: CacheEntry[] }[] = [];
    const now = Date.now();
    const sortedFreqs = Array.from(this.freqBuckets.keys()).sort((a, b) => a - b);

    for (const freq of sortedFreqs) {
      const bucket = this.freqBuckets.get(freq);
      if (!bucket) continue;

      const entries: CacheEntry[] = [];
      let curr = bucket.head.next;
      while (curr && curr !== bucket.tail) {
        if (!this.isExpired(curr, now)) {
          entries.push({
            key: curr.key,
            value: curr.value,
            freq: curr.freq,
            expiresAt: curr.expiresAt,
            createdAt: curr.createdAt,
          });
        }
        curr = curr.next;
      }

      if (entries.length > 0) {
        result.push({ freq, entries });
      }
    }

    return result;
  }

  public getAllEntries(): CacheEntry[] {
    return Array.from(this.items.values())
      .filter((n) => !this.isExpired(n))
      .map((n) => ({
        key: n.key,
        value: n.value,
        freq: n.freq,
        expiresAt: n.expiresAt,
        createdAt: n.createdAt,
      }));
  }
}
