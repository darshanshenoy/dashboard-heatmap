import React, { useEffect, useState } from "react";
import PropTypes from "prop-types";
import "./Heatmap.css";

const Heatmap = ({ symbols, sizes, colors }) => {
  const [displayData, setDisplayData] = useState([]);

  useEffect(() => {
    if (symbols.length === 0) return;

    // Smooth transition of data
    setDisplayData(prevData => {
      return symbols.map((symbol, index) => {
        const existing = prevData.find(d => d.symbol === symbol);
        return {
          symbol,
          size: existing ? existing.size * 0.7 + sizes[index] * 0.3 : sizes[index], // Smooth size change
          color: existing ? existing.color * 0.7 + colors[index] * 0.3 : colors[index] // Smooth color transition
        };
      });
    });
  }, [symbols, sizes, colors]);

  if (!displayData.length) {
    return <p>Loading data...</p>;
  }

  const maxSize = Math.max(...displayData.map(d => d.size));
  const minSize = Math.min(...displayData.map(d => d.size));

  return (
    <div className="heatmap-grid">
      {displayData.map(({ symbol, size, color }) => {
        const normalizedSize = ((size - minSize) / (maxSize - minSize)) * 200 + 50;
        const bgColor = color >= 0 
          ? `rgba(0, 200, 0, ${0.5 + Math.min(color / 10, 0.5)})`
          : `rgba(200, 0, 0, ${0.5 + Math.min(Math.abs(color) / 10, 0.5)})`;

        return (
          <div
            key={symbol}
            className="heatmap-block"
            style={{
              width: `${normalizedSize}px`,
              height: `${normalizedSize}px`,
              backgroundColor: bgColor,
              transition: "all 0.5s ease-in-out", // Smooth animations
            }}
          >
            <span className="heatmap-label">{symbol}</span>
            <span className="heatmap-value">{color.toFixed(2)}%</span>
          </div>
        );
      })}
    </div>
  );
};

Heatmap.propTypes = {
  symbols: PropTypes.array.isRequired,
  sizes: PropTypes.array.isRequired,
  colors: PropTypes.array.isRequired,
};

export default Heatmap;
