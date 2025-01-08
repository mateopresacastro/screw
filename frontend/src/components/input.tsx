import * as React from "react";

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<"input">>(
  ({ type, ...props }, ref) => {
    return (
      <div className="inline-block w-full">
        <label className="py-2 px-4 border border-gray-500 bg-gray-300 hover:bg-gray-400 cursor-pointer  w-full flex">
          <p className="mx-auto">Choose Files</p>
          <input type={type} className="hidden" ref={ref} {...props} />
        </label>
      </div>
    );
  }
);

Input.displayName = "Input";

export { Input };
