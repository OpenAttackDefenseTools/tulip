import classNames from "classnames";

export interface RadioGroupProps {
    options: string[];
    value: string;
    className: string;
    onChange: (option: string) => void;
}

export function RadioGroup(props: RadioGroupProps) {
    return (
        <div className={props.className}>
            {props.options.map((option) => (
                <div
                    key={option}
                    onClick={() => props.onChange(option)}
                    className={classNames({
                        "bg-gray-200": option === props.value,
                        "px-1 rounded-sm": true,
                    })}
                >
                    {option}
                </div>
            ))}
        </div>
    );
}