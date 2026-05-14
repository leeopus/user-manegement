import { ImageResponse } from "next/og"

export const size = { width: 32, height: 32 }
export const contentType = "image/png"

export default function Icon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: 32,
          height: 32,
          borderRadius: 8,
          background: "#3b82f6",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          position: "relative",
        }}
      >
        <div
          style={{
            width: 18,
            height: 18,
            borderRadius: 9,
            border: "3px solid white",
            background: "transparent",
          }}
        />
        <div
          style={{
            position: "absolute",
            top: 4.5,
            right: 4.5,
            width: 5,
            height: 5,
            background: "white",
            borderRadius: 1,
            transform: "rotate(45deg)",
          }}
        />
      </div>
    ),
    { ...size }
  )
}
