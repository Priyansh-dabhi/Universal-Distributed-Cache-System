import React, { useEffect } from 'react';
import { motion, useSpring, useTransform } from 'framer-motion';

interface AnimatedCounterProps {
  value: number;
  decimals?: number;
  prefix?: string;
  suffix?: string;
  className?: string;
}

export const AnimatedCounter: React.FC<AnimatedCounterProps> = ({
  value,
  decimals = 0,
  prefix = '',
  suffix = '',
  className,
}) => {
  const spring = useSpring(value, {
    stiffness: 80,
    damping: 18,
    mass: 0.8,
  });

  const display = useTransform(spring, (latest) => {
    return latest.toLocaleString('en-US', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });
  });

  useEffect(() => {
    spring.set(value);
  }, [value, spring]);

  return (
    <span className={className}>
      {prefix}
      <motion.span>{display}</motion.span>
      {suffix}
    </span>
  );
};
