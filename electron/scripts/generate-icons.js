#!/usr/bin/env node

/**
 * Icon Generator for Swarm Desktop
 * 
 * Generates application icons in all required formats:
 * - PNG icons at standard sizes (16, 32, 48, 64, 128, 256, 512, 1024)
 * - ICO file for Windows (contains multiple sizes)
 * - ICNS file for macOS (requires external tool - generates PNGs for manual conversion)
 * 
 * Design: Stylized "S" lettermark on a gradient amber/gold background
 * representing a swarm/hive motif.
 */

const sharp = require('sharp');
const path = require('path');
const fs = require('fs');

const BUILD_DIR = path.join(__dirname, '..', 'build');
const ICONS_DIR = path.join(BUILD_DIR, 'icons');

// Icon sizes needed for various platforms
const SIZES = [16, 32, 48, 64, 128, 256, 512, 1024];

// Create an SVG icon with a stylized "S" on gradient background
function createIconSvg(size) {
  // Scale factors
  const padding = size * 0.1;
  const innerSize = size - (padding * 2);
  const centerX = size / 2;
  const centerY = size / 2;
  const cornerRadius = size * 0.18;
  
  // S path - stylized with flow suggesting swarm movement
  const sScale = innerSize / 100;
  const sOffsetX = padding;
  const sOffsetY = padding;
  
  // Create SVG with gradient background and stylized S
  return `<?xml version="1.0" encoding="UTF-8"?>
<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <!-- Background gradient - amber/gold for hive/swarm theme -->
    <linearGradient id="bgGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#F59E0B"/>
      <stop offset="50%" style="stop-color:#D97706"/>
      <stop offset="100%" style="stop-color:#B45309"/>
    </linearGradient>
    <!-- Subtle inner shadow -->
    <filter id="innerShadow" x="-50%" y="-50%" width="200%" height="200%">
      <feDropShadow dx="0" dy="${size * 0.02}" stdDeviation="${size * 0.03}" flood-color="#000" flood-opacity="0.3"/>
    </filter>
  </defs>
  
  <!-- Rounded rectangle background -->
  <rect x="${padding/2}" y="${padding/2}" width="${size - padding}" height="${size - padding}" 
        rx="${cornerRadius}" ry="${cornerRadius}" 
        fill="url(#bgGrad)"/>
  
  <!-- Stylized S letterform -->
  <g transform="translate(${centerX}, ${centerY}) scale(${sScale * 0.65})" filter="url(#innerShadow)">
    <path d="M 15 -40 
             C 35 -40, 45 -30, 45 -15
             C 45 0, 30 8, 0 12
             C -30 16, -45 24, -45 38
             C -45 52, -30 60, -5 60
             C 15 60, 30 55, 40 45"
          fill="none" 
          stroke="white" 
          stroke-width="16" 
          stroke-linecap="round"
          stroke-linejoin="round"/>
  </g>
  
  <!-- Small dots suggesting swarm particles -->
  <circle cx="${centerX - size * 0.28}" cy="${centerY - size * 0.22}" r="${Math.max(1, size * 0.025)}" fill="rgba(255,255,255,0.6)"/>
  <circle cx="${centerX + size * 0.3}" cy="${centerY + size * 0.18}" r="${Math.max(1, size * 0.02)}" fill="rgba(255,255,255,0.5)"/>
  <circle cx="${centerX - size * 0.2}" cy="${centerY + size * 0.3}" r="${Math.max(1, size * 0.015)}" fill="rgba(255,255,255,0.4)"/>
</svg>`;
}

async function generatePng(size, outputPath) {
  const svg = createIconSvg(size);
  await sharp(Buffer.from(svg))
    .resize(size, size)
    .png()
    .toFile(outputPath);
  console.log(`Generated: ${outputPath}`);
}

async function generateIco(sizes, outputPath) {
  // Use CLI approach since the module uses ES exports
  const { execSync } = require('child_process');
  
  // Generate temporary PNGs for ICO
  const tempPngs = [];
  for (const size of sizes) {
    const tempPath = path.join(BUILD_DIR, `temp_${size}.png`);
    await generatePng(size, tempPath);
    tempPngs.push(tempPath);
  }
  
  // Create ICO from the 256px PNG using CLI (it auto-generates all sizes)
  const png256 = tempPngs.find(p => p.includes('256'));
  execSync(`npx png-to-ico "${png256}" > "${outputPath}"`, { shell: true });
  console.log(`Generated: ${outputPath}`);
  
  // Clean up temp files
  for (const tempPath of tempPngs) {
    fs.unlinkSync(tempPath);
  }
}

async function main() {
  // Ensure directories exist
  if (!fs.existsSync(BUILD_DIR)) {
    fs.mkdirSync(BUILD_DIR, { recursive: true });
  }
  if (!fs.existsSync(ICONS_DIR)) {
    fs.mkdirSync(ICONS_DIR, { recursive: true });
  }
  
  console.log('Generating Swarm Desktop icons...\n');
  
  // Generate PNG icons at all sizes
  for (const size of SIZES) {
    await generatePng(size, path.join(ICONS_DIR, `${size}x${size}.png`));
  }
  
  // Generate main icon.png (512x512 for Linux)
  await generatePng(512, path.join(BUILD_DIR, 'icon.png'));
  
  // Generate ICO for Windows (16, 32, 48, 64, 128, 256)
  const icoSizes = [16, 32, 48, 64, 128, 256];
  await generateIco(icoSizes, path.join(BUILD_DIR, 'icon.ico'));
  
  console.log('\n✓ Icon generation complete!');
  console.log('\nNote: For macOS .icns file, use one of these methods:');
  console.log('  1. On macOS: iconutil -c icns build/icons.iconset');
  console.log('  2. Use an online converter with the generated PNGs');
  console.log('  3. electron-builder will auto-convert icon.png to icns on macOS\n');
}

main().catch(console.error);
