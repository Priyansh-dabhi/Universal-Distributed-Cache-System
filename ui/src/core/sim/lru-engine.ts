import type { CacheEntry } from '../types';

interface LRUNode {
  key: string;
  value: string;
  expiresAt: number | null; // ms timestamp or null
  createdAt: number;
  prev: LRUNode | null;
  next: LRUNode | null;
}

/**
 * LRU (Least Recently Used) in-memory cache engine.
 * Direct TypeScript port of `internal/cache/lru.go`.
 * Uses a doubly linked list with sentinel head and tail.
 */
export class LRUEngine {
  private capacity: number;
  private items: Map<string, LRUNode> = new Map();
  private head: LRUNode; // sentinel head: head.next is MRU
  private tail: LRUNode; // sentinel tail: tail.prev is LRU

  constructor(capacity: number) {
    if (capacity <= 0) {
      throw new Error('Capacity must be greater than 0');
    }
    this.capacity = capacity;

    this.head = {
      key: '',
      value: '',
      expiresAt: null,
      createdAt: 0,
      prev: null,
      next: null,
    };
    this.tail = {
      key: '',
      value: '',
      expiresAt: null,
      createdAt: 0,
      prev: null,
      next: null,
    };

    this.head.next = this.tail;
    this.tail.prev = this.head;
  }

  public getCapacity(): number {
    return this.capacity;
  }

  public size(): number {
    return this.items.size;
  }

  private isExpired(node: LRUNode, now: number = Date.now()): boolean {
    return node.expiresAt !== null && now >= node.expiresAt;
  }

  private addToFront(node: LRUNode): void {
    node.prev = this.head;
    node.next = this.head.next;
    if (this.head.next) {
      this.head.next.prev = node;
    }
    this.head.next = node;
  }

  private removeNode(node: LRUNode): void {
    if (node.prev) {
      node.prev.next = node.next;
    }
    if (node.next) {
      node.next.prev = node.prev;
    }
    node.prev = null;
    node.next = null;
  }

  private moveToFront(node: LRUNode): void {
    this.removeNode(node);
    this.addToFront(node);
  }

  private removeLRU(): LRUNode | null {
    if (this.tail.prev === this.head || !this.tail.prev) {
      return null;
    }
    const lru = this.tail.prev;
    this.removeNode(lru);
    return lru;
  }

  /**
   * Set key-value pair.
   * Returns: { evicted: boolean; evictedKey?: string }
   */
  public set(
    key: string,
    value: string,
    ttlMs?: number | null
  ): { evicted: boolean; evictedKey?: string } {
    const expiresAt = ttlMs && ttlMs > 0 ? Date.now() + ttlMs : null;

    // Check if key already exists
    const existing = this.items.get(key);
    if (existing) {
      existing.value = value;
      existing.expiresAt = expiresAt;
      this.moveToFront(existing);
      return { evicted: false };
    }

    let evicted = false;
    let evictedKey: string | undefined;

    // Evict least recently used if at capacity
    if (this.items.size >= this.capacity) {
      const lru = this.removeLRU();
      if (lru) {
        this.items.delete(lru.key);
        evicted = true;
        evictedKey = lru.key;
      }
    }

    const newNode: LRUNode = {
      key,
      value,
      expiresAt,
      createdAt: Date.now(),
      prev: null,
      next: null,
    };

    this.addToFront(newNode);
    this.items.set(key, newNode);

    return { evicted, evictedKey };
  }

  /**
   * Get value by key.
   * Returns: { value: string | null; found: boolean; expired: boolean }
   */
  public get(key: string): { value: string | null; found: boolean; expired: boolean } {
    const node = this.items.get(key);
    if (!node) {
      return { value: null, found: false, expired: false };
    }

    // Lazy expiration
    if (this.isExpired(node)) {
      this.removeNode(node);
      this.items.delete(key);
      return { value: null, found: false, expired: true };
    }

    this.moveToFront(node);
    return { value: node.value, found: true, expired: false };
  }

  /**
   * Delete key.
   * Returns: { deleted: boolean; expired: boolean }
   */
  public delete(key: string): { deleted: boolean; expired: boolean } {
    const node = this.items.get(key);
    if (!node) {
      return { deleted: false, expired: false };
    }

    const expired = this.isExpired(node);
    this.removeNode(node);
    this.items.delete(key);

    return { deleted: !expired, expired };
  }

  public clear(): void {
    this.items.clear();
    this.head.next = this.tail;
    this.tail.prev = this.head;
  }

  /**
   * Snapshot of entries ordered from MRU (index 0) to LRU (tail).
   * Used directly by the LRU visualizer.
   */
  public getOrderedEntries(): CacheEntry[] {
    const result: CacheEntry[] = [];
    const now = Date.now();
    let curr = this.head.next;

    while (curr && curr !== this.tail) {
      if (!this.isExpired(curr, now)) {
        result.push({
          key: curr.key,
          value: curr.value,
          expiresAt: curr.expiresAt,
          createdAt: curr.createdAt,
        });
      }
      curr = curr.next;
    }

    return result;
  }
}
