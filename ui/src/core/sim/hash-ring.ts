import { fnv1a, formatHashHex, hashToAngle } from './fnv1a';
import type { RingNode, RingPosition, RouteResult } from '../types';

export const DEFAULT_REPLICAS = 50; // Optimized for smooth SVG rendering

export interface HashRingConfig {
  replicas?: number;
}

export class ConsistentHashRing {
  private replicas: number;
  private nodes: Map<string, RingNode> = new Map();
  private ringPositions: number[] = [];
  private ringMap: Map<number, { node: RingNode; token: string }> = new Map();

  constructor(replicas: number = DEFAULT_REPLICAS) {
    this.replicas = replicas > 0 ? replicas : DEFAULT_REPLICAS;
  }

  public getReplicas(): number {
    return this.replicas;
  }

  public setReplicas(replicas: number): void {
    this.replicas = replicas > 0 ? replicas : DEFAULT_REPLICAS;
    this.rebuildRing();
  }

  public addNode(node: RingNode): void {
    if (this.nodes.has(node.id)) {
      throw new Error(`Node ${node.id} already exists in hash ring`);
    }
    this.nodes.set(node.id, { ...node });
    this.rebuildRing();
  }

  public removeNode(nodeId: string): void {
    if (!this.nodes.has(nodeId)) {
      throw new Error(`Node ${nodeId} not found in hash ring`);
    }
    this.nodes.delete(nodeId);
    this.rebuildRing();
  }

  public getNodes(): RingNode[] {
    return Array.from(this.nodes.values()).sort((a, b) => a.id.localeCompare(b.id));
  }

  public getNodeCount(): number {
    return this.nodes.size;
  }

  public getPositions(): RingPosition[] {
    return this.ringPositions.map((pos) => {
      const entry = this.ringMap.get(pos)!;
      return {
        position: pos,
        angle: hashToAngle(pos),
        nodeId: entry.node.id,
        token: entry.token,
        isVirtual: entry.token.includes('#'),
      };
    });
  }

  /**
   * Route key to nearest clockwise node using binary search (O(log R)).
   */
  public route(key: string): RouteResult {
    if (!key || key.trim() === '') {
      throw new Error('Key must not be empty');
    }

    if (this.ringPositions.length === 0) {
      throw new Error('No cache nodes available on ring');
    }

    const keyHash = fnv1a(key);

    // Binary search for first position >= keyHash
    let low = 0;
    let high = this.ringPositions.length - 1;
    let targetIdx = 0;

    if (keyHash > this.ringPositions[high]) {
      // Wrap around to position 0
      targetIdx = 0;
    } else {
      while (low <= high) {
        const mid = Math.floor((low + high) / 2);
        if (this.ringPositions[mid] >= keyHash) {
          targetIdx = mid;
          high = mid - 1;
        } else {
          low = mid + 1;
        }
      }
    }

    const targetPos = this.ringPositions[targetIdx];
    const targetEntry = this.ringMap.get(targetPos)!;

    return {
      key,
      hash: keyHash,
      hashHex: formatHashHex(keyHash),
      angle: hashToAngle(keyHash),
      targetNodeId: targetEntry.node.id,
      targetAddress: targetEntry.node.address,
      targetReplicaToken: targetEntry.token,
      targetPosition: targetPos,
    };
  }

  private rebuildRing(): void {
    this.ringPositions = [];
    this.ringMap.clear();

    if (this.nodes.size === 0) {
      return;
    }

    const sortedNodeIds = Array.from(this.nodes.keys()).sort();

    for (const nodeId of sortedNodeIds) {
      const node = this.nodes.get(nodeId)!;
      for (let i = 0; i < this.replicas; i++) {
        const token = `${nodeId}#${i}`;
        let pos = fnv1a(token);

        // Linear probing for collision avoidance (matches ring.go)
        while (this.ringMap.has(pos)) {
          pos = (pos + 1) >>> 0;
        }

        this.ringMap.set(pos, { node, token });
        this.ringPositions.push(pos);
      }
    }

    // Sort positions ascending for binary search
    this.ringPositions.sort((a, b) => a - b);
  }
}
