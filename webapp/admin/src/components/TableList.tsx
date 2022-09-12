import { FC } from "react";

interface tableListProps {
  list: any[];
}

export const TableList: FC<tableListProps> = ({ list }) => (
  <>
    {list.length === 0 ? (
      <span>データなし</span>
    ) : (
      <table className="table-list">
        <thead>
          <tr>
            {Object.keys(list[0]).map(key => (
              <th key={key}>{key}</th>
            ))}
          </tr>
        </thead>

        <tbody>
          {list.map(item => (
            <tr>
              {Object.values(item).map(val => (
                <td>{val as any}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    )}
  </>
);
