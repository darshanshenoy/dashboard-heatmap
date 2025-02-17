import React, { useEffect, useState } from "react";
import axios from "axios";
import Heatmap from "./Heatmap";

const Dashboard = () => {
  const [data, setData] = useState([]);
  const [symbols, setSymbols] = useState([]);
  const [heatmapData, setHeatmapData] = useState({ sizes: [], colors: [] });

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await axios.get("http://104.199.197.142:8080/ohlcv");

        if (!Array.isArray(response.data) || response.data.length === 0) {
          console.warn("API returned empty or invalid data.");
          setData([]);
          return;
        }

        const processedData = response.data.map(item => ({
          symbol: item.symbol,
          price: parseFloat(item.close),
          marketCap: parseFloat(item.volume) * parseFloat(item.close), // Approx. Market Cap
          priceChange: ((parseFloat(item.close) - parseFloat(item.open)) / parseFloat(item.open)) * 100
        }));

        setData(processedData);
      } catch (error) {
        console.error("Error fetching data:", error);
      }
    };

    fetchData();

    const interval = setInterval(fetchData, 5000); // Auto-refresh every 5 sec
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (data.length === 0) {
      setSymbols([]);
      setHeatmapData({ sizes: [], colors: [] });
      return;
    }

    const sortedData = [...data].sort((a, b) => b.marketCap - a.marketCap);

    setSymbols(sortedData.map(item => item.symbol));
    setHeatmapData({
      sizes: sortedData.map(item => item.marketCap),
      colors: sortedData.map(item => item.priceChange),
    });

  }, [data]);

  return (
    <div>
      <h1>Crypto Market Performance</h1>
      <Heatmap symbols={symbols} sizes={heatmapData.sizes} colors={heatmapData.colors} />
    </div>
  );
};

export default Dashboard;
